package activedirectory

import (
	"crypto/tls"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/go-ldap/ldap/v3"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

type ADClient struct {
	logger        hclog.Logger
	config        Config
	conn          *ldap.Conn
	mux           sync.Mutex
	activeWorkers int
}

type Config struct {
	serverURL   string
	domain      string
	topDN       string
	username    string
	password    string
	insecureTLS bool
}

// initialiseConn will start AD connection and bind with given username. it will also keep tract of number of workers using connection.
// if connection is already active / open it will return.
// ldap is an async communication, means one connection can be used to send multiple messages.
func (c *ADClient) initialiseConn() error {
	var err error

	c.mux.Lock()
	defer c.mux.Unlock()

	c.activeWorkers++

	if c.conn != nil && !c.conn.IsClosing() {
		c.logger.Debug("ADClient.initialiseConn: connection is active", "activeWorkers", c.activeWorkers)
		return nil
	}

	c.logger.Debug("ADClient.initialiseConn: initiating AD connection", "URL", c.config.serverURL, "insecureTLS", c.config.insecureTLS, "activeWorkers", c.activeWorkers)
	dialOpt := ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: c.config.insecureTLS})
	c.conn, err = ldap.DialURL(c.config.serverURL, dialOpt)
	if err != nil {
		return fmt.Errorf("ADClient.initialiseConn: unable to connect to ad server err:%w", err)
	}

	_, ok := c.conn.TLSConnectionState()
	c.logger.Debug("ADClient.initialiseConn: TLS Connection state", "TLS", ok)

	// Bind AD user for LDAP operations
	c.logger.Debug("ADClient.initialiseConn: initiating AD User Bind", "username", c.config.username)
	if err := c.conn.Bind(c.config.username, c.config.password); err != nil {
		c.conn.Close()
		return fmt.Errorf("ADClient.initialiseConn: AD Bind user err: %w", err)
	}

	return nil
}

// done will keep track of active worker and close connection when no one is using it.
func (c *ADClient) done() {
	c.mux.Lock()
	defer c.mux.Unlock()

	c.activeWorkers--
	c.logger.Debug("ADClient.done", "activeWorkers", c.activeWorkers)
	if c.activeWorkers == 0 {
		c.logger.Debug("ADClient.done: there are no active workers remaining closing AD connection")
		c.conn.Close()
	}
}

func getObjectByDN(conn *ldap.Conn, dn string) (*ldap.Entry, error) {
	sReq := &ldap.SearchRequest{
		BaseDN:       dn,
		Scope:        ldap.ScopeWholeSubtree,
		DerefAliases: ldap.NeverDerefAliases,
		SizeLimit:    0,
		TimeLimit:    0,
		TypesOnly:    false,
		Filter:       "(objectClass=*)",
		Attributes:   []string{"*"},
		Controls:     nil,
	}

	sr, err := conn.Search(sReq)
	if err != nil {
		if ldap.IsErrorWithCode(err, 32) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	for _, e := range sr.Entries {
		if strings.EqualFold(e.DN, dn) {
			return e, nil
		}
	}

	return nil, ErrObjectNotFound
}

func getObjectByID(c *ADClient, id string) (*ldap.Entry, error) {

	sReq := &ldap.SearchRequest{
		BaseDN:       c.config.topDN,
		Scope:        ldap.ScopeWholeSubtree,
		DerefAliases: ldap.NeverDerefAliases,
		SizeLimit:    0,
		TimeLimit:    0,
		TypesOnly:    false,
		Filter:       "(objectGUID=" + parseID(id) + ")",
		Attributes:   []string{"*"},
		Controls:     nil,
	}

	sr, err := c.conn.Search(sReq)
	if err != nil {
		if ldap.IsErrorWithCode(err, 32) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	if len(sr.Entries) == 0 {
		return nil, ErrObjectNotFound
	}

	if len(sr.Entries) > 1 {
		return nil, fmt.Errorf("multiple ldap object found for GUID: %s", id)
	}
	return sr.Entries[0], nil
}

func getObjectsBySAM(c *ADClient, sam string) ([]*ldap.Entry, error) {
	sReq := &ldap.SearchRequest{
		BaseDN:       c.config.topDN,
		Scope:        ldap.ScopeWholeSubtree,
		DerefAliases: ldap.NeverDerefAliases,
		SizeLimit:    0,
		TimeLimit:    0,
		TypesOnly:    false,
		Filter:       "(sAMAccountName=" + sam + ")",
		Attributes:   []string{"*"},
		Controls:     nil,
	}

	sr, err := c.conn.Search(sReq)
	if err != nil {
		if ldap.IsErrorWithCode(err, 32) {
			return nil, ErrObjectNotFound
		}
		return nil, err
	}

	return sr.Entries, nil
}

func addObject(conn *ldap.Conn, addReq *ldap.AddRequest) (string, error) {
	if err := conn.Add(addReq); err != nil {
		return "", fmt.Errorf("addObject: unable to add object dn:%v to LDAP server err:%w", addReq.DN, err)
	}

	e, err := getObjectByDN(conn, addReq.DN)
	if err != nil {
		return "", fmt.Errorf("addObject: unable to get created object's ID  dn:%v err:%w", addReq.DN, err)
	}

	rawGuid := e.GetRawAttributeValue("objectGUID")
	guid, err := decodeGUID(rawGuid)
	if err != nil {
		return "", fmt.Errorf("addObject: unable to convert raw GUID to string   rawGUID:%x err:%w", rawGuid, err)
	}
	return guid, nil
}

// func resourceExistsObject(d *schema.ResourceData, m interface{}) (bool, error) {
// 	c := m.(*ADClient)
// 	err := c.initialiseConn()
// 	if err != nil {
// 		return false, fmt.Errorf("resourceExistsObject: unable to connect to LDAP server err:%w", err)
// 	}
// 	defer c.done()

// 	id, err := encodeGUID(d.Id())
// 	if err != nil {
// 		return false, fmt.Errorf("resourceExistsObject: unable to encode GUID:%v err:%w", d.Id(), err)
// 	}
// 	e, err := getObjectByID(c, id)
// 	if err != nil {
// 		if errors.Is(err, ErrObjectNotFound) {
// 			return false, nil
// 		}
// 		return false, fmt.Errorf("resourceExistsObject: unable to search user with ID  GUID:%v err:%w", d.Id(), err)
// 	}
// 	c.logger.Debug("resourceExistsObject: user object found", "dn", e.DN)
// 	return true, nil
// }

func resourceDeleteObject(d *schema.ResourceData, meta interface{}) error {
	var err error
	c := meta.(*ADClient)
	err = c.initialiseConn()
	if err != nil {
		return fmt.Errorf("resourceDeleteObject: unable to connect to LDAP server err:%w", err)
	}
	defer c.done()

	id, err := encodeGUID(d.Id())
	if err != nil {
		return fmt.Errorf("resourceDeleteObject: unable to encode GUID:%v err:%w", d.Id(), err)
	}
	e, err := getObjectByID(c, id)
	if err != nil {
		if errors.Is(err, ErrObjectNotFound) {
			return nil
		}
		return fmt.Errorf("resourceDeleteObject: unable to search object with ID GUID:%v err:%w", d.Id(), err)
	}

	request := &ldap.DelRequest{DN: e.DN}
	err = c.conn.Del(request)
	if err != nil {
		return fmt.Errorf("resourceDeleteObject: unable to delete object guid:%v dn:%v err:%w", d.Id(), e.DN, err)
	}
	c.logger.Info("resourceDeleteObject: AD object deleted", "guid", d.Id(), "dn", e.DN)
	return nil
}

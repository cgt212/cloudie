package main

import "fmt"
import "github.com/aws/aws-sdk-go/service/ec2"
import "github.com/jackc/pgx"

//import log "gopkg.in/inconshreveable/log15.v2"
import "os"
import "sync"

type Db struct {
	Connection *pgx.ConnPool
	lock       *sync.Mutex
}

func createConfig() pgx.ConnConfig {
	var config pgx.ConnConfig

	config.Host = os.Getenv("CLOUDIE_DB_HOST")
	if config.Host == "" {
		config.Host = "localhost"
	}
	config.User = os.Getenv("CLOUDIE_DB_USER")
	if config.User == "" {
		config.User = os.Getenv("USER")
	}
	config.Database = os.Getenv("CLOUDIE_DB_NAME")
	if config.Database == "" {
		config.Database = "aws"
	}
	config.Password = os.Getenv("CLOUDIE_DB_PASSWORD")
	if config.Password == "" {
		config.Password = "cl0udation"
	}

	// Set up some logging
	//logger := log.New("cloudie", "database")
	//config.Logger = logger
	//config.LogLevel, _ = pgx.LogLevelFromString("error")

	return config
}

func DbConnect(maxConn int) (*Db, error) {
	connConf := createConfig()
	conf := pgx.ConnPoolConfig{ConnConfig: connConf, MaxConnections: maxConn, AfterConnect: PrepareStatements}
	conn, err := pgx.NewConnPool(conf)
	if err != nil {
		fmt.Printf("Unable to connect to database: %s\n", err)
		return nil, err
	}
	var ret Db
	ret.Connection = conn
	ret.lock = &sync.Mutex{}
	if err != nil {
		fmt.Printf("Unable to prepare statements")
	}
	return &ret, nil
}

func PrepareStatements(d *pgx.Conn) error {
	_, err := d.Prepare("insertInstance", "INSERT INTO ec2_instances VALUES($1, $2, NOW())")
	if err != nil {
		return err
	}
	_, err = d.Prepare("updateTimestamp", "UPDATE ec2_instances SET last_updated = NOW() WHERE id=$1")
	if err != nil {
		return err
	}
	_, err = d.Prepare("updateData", "UPDATE ec2_instances SET data = $1, last_updated = NOW() WHERE id=$2")
	if err != nil {
		return err
	}
	_, err = d.Prepare("getInstanceById", "SELECT data FROM ec2_instances WHERE id=$1")
	if err != nil {
		return err
	}
	return err
}

func (d *Db) InsertInstance(inst *ec2.Instance) (int, error) {
	//result, err := d.Connection.Exec("Insert into ec2_instances values($1, $2, NOW())", *inst.InstanceId, inst)
	d.lock.Lock()
	result, err := d.Connection.Exec("insertInstance", *inst.InstanceId, inst)
	if err != nil {
		fmt.Printf("Error inserting into the database: %s\n", err)
	}
	ret := int(result.RowsAffected())
	d.lock.Unlock()
	return ret, err
}

func (d *Db) UpdateInstanceTimestamp(id string) (int, error) {
	d.lock.Lock()
	result, err := d.Connection.Exec("updateTimestamp", id)
	if err != nil {
		fmt.Printf("Error updating timestamp for instance %s: %s\n", id, err)
	}
	ret := int(result.RowsAffected())
	d.lock.Unlock()
	return ret, err
}

func (d *Db) UpdateInstanceData(instance *ec2.Instance) (int, error) {
	d.lock.Lock()
	result, err := d.Connection.Exec("updateData", instance, *instance.InstanceId)
	if err != nil {
		fmt.Printf("Error updating instance data for %s: %s\n", *instance.InstanceId, err)
	}
	ret := int(result.RowsAffected())
	d.lock.Unlock()
	return ret, err
}

func (d *Db) GetInstanceById(id string) (*ec2.Instance, error) {
	var i ec2.Instance
	d.lock.Lock()
	defer d.lock.Unlock()
	err := d.Connection.QueryRow("getInstanceById", id).Scan(&i)
	if err != nil {
		fmt.Printf("Error getting row for id (%s): %s\n", id, err)
		return nil, err
	}

	return &i, err
}

func (d *Db) DeleteOldInstances() (int, error) {
	d.lock.Lock()
	result, err := d.Connection.Exec("DELETE FROM ec2_instances WHERE last_updated < now() - interval '5 minute'")
	d.lock.Unlock()
	if err != nil {
		fmt.Printf("Error deleting old entries: %s\n", err)
		return 0, err
	}
	ret := int(result.RowsAffected())

	return ret, nil
}

func (d *Db) Close() {
	d.Close()
}

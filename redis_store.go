package main

import (
    "errors"
    "log"
    
    "menteslibres.net/gosexy/redis"
)

const ClientsKey = "SocketClients"

type RedisStore struct {
    clientsKey  string

    server      string
    port        uint
    pool        redisPool
}

//connection pool implimentation
type redisPool struct {
    connections []*redis.Client
    maxIdle     int
    connFn      func() (*redis.Client, error) // function to create new connection.
}

func (this *redisPool) Get() (*redis.Client, bool) {
    log.Println(len(this.connections))
    if(len(this.connections) == 0) {
        conn, err := this.connFn()
        if err != nil {
            return nil, false
        }
        
        return conn, true
    }
    
    var conn *redis.Client
    conn, this.connections = this.connections[len(this.connections)-1], this.connections[:len(this.connections)-1]
    if err := this.testConn(conn); err != nil {
        return this.Get() // if connection is bad, get the next one in line until base case is hit, then create new client
    }
    
    return conn, true
}

func (this *redisPool) Close(conn *redis.Client) {
    if(len(this.connections) < this.maxIdle) {
        this.connections = append(this.connections, conn)
        return
    }
    
    conn.Quit()
}

func (this *redisPool) testConn(conn *redis.Client) error {
    if _, err := conn.Ping(); err != nil {
        return err
    }
    
    return nil
}

func (this *RedisStore) GetConn() (*redis.Client, error) {
    
    client, ok := this.pool.Get()
    if !ok {
        return nil, errors.New("Error while getting redis connection")
    }
    
    return client, nil
    
}

func (this *RedisStore) CloseConn(conn *redis.Client) {
    this.pool.Close(conn)
}

func (this *RedisStore) Subscribe(c chan []string, channel string) (*redis.Client, error) {
    consumer := redis.New()
    err := consumer.ConnectNonBlock(this.server, this.port)
    if err != nil {
        return nil, err
    }
    
    go consumer.Subscribe(c, channel)
    return consumer, nil
}

func (this *RedisStore) Publish(channel string, message string) {
    publisher := redis.New()
    err       := publisher.Connect(this.server, this.port)
    if err != nil {
        return
    }
    
    publisher.Publish(channel, message)
    
    publisher.Quit()
}

func (this *RedisStore) Save(UID string) (error) {
    client, err := this.GetConn()
    if(err != nil) {
        return err
    }
    defer this.CloseConn(client)
 
    _, err = client.SAdd(this.clientsKey, UID)
    if err != nil {
        log.Printf("%s\n", err.Error())
    }
    
    return nil
}

func (this *RedisStore) Remove(UID string) (error) {
    client, err := this.GetConn()
    if(err != nil) {
        return err
    }
    defer this.CloseConn(client)
 
    _, err = client.SRem(this.clientsKey, UID)
    if err != nil {
        return err
    }
    
    return nil
}

func (this *RedisStore) Clients() ([]string, error) {
    client, err := this.GetConn()
    if(err != nil) {
        return nil, err
    }
    defer this.CloseConn(client)
 
    socks, err1 := client.SMembers(this.clientsKey)
    if err1 != nil {
        return nil, err1
    }
    
    return socks, nil
}

func (this *RedisStore) Count() (int64, error) {
    client, err := this.GetConn()
    if(err != nil) {
        return 0, err
    }
    defer this.CloseConn(client)
 
    socks, err1 := client.SCard(this.clientsKey)
    if err1 != nil {
        return 0, err1
    }
    
    return socks, nil
}
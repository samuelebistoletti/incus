package main

import (
    "log"
    "encoding/json"
    "errors"
    "time"
)

type Message struct {
    Event string
    Body map[string]interface{}
    Time int64
}

func (this *Message) FromSocket(sock *Socket) {
    if DEBUG { log.Printf("Handling socket message of type %s\n", this.Event) }
    
    switch this.Event {
    case "MessageUser":
        if(!CLIENT_BROAD) { return }
        
        if(sock.Server.Store.StorageType == "redis") {
            this.forwardToRedis(sock.Server)
            return 
        }
        
        this.messageUser(sock.Server)
        
    case "MessageAll":
        if(!CLIENT_BROAD) { return }
    
        if(sock.Server.Store.StorageType == "redis") {
            this.forwardToRedis(sock.Server)
            return
        }
        
        this.messageAll(sock.Server)
    case "SetPage":
        page, ok := this.Body["Page"].(string)
        if !ok || page == "" {
            return
        }

        if sock.Page != "" {
            sock.Server.Store.UnsetPage(sock)  //remove old page if it exists
        }
        
        sock.Page = page
        sock.Server.Store.SetPage(sock) // set new page
    }
}

func (this *Message) FromRedis(server *Server) {
    if DEBUG { log.Printf("Handling redis message of type %s\n", this.Event) }
    
    switch this.Event {
    
    case "MessageUser":
        this.messageUser(server)
    
    case "MessageAll":
        this.messageAll(server)

    case "MessagePage": 
        msg, err := this.formatBody()
        if err != nil {
            return
        }

        page, ok := this.Body["Page"].(string)
        if !ok {
            return
        }

        pageMap := server.Store.getPage(page)
        if pageMap == nil {
            return
        }
        
        for _, sock := range pageMap {
            sock.buff <- msg
        }

        return
    }
}

func (this *Message) formatBody() (*Message, error) {    
    event, e_ok := this.Body["Event"].(string)
    body,  b_ok := this.Body["Message"].(map[string]interface{})
    
    if !b_ok || ! e_ok {
        return nil, errors.New("Could not format message body")
    }
    
    msg := &Message{event, body, time.Now().UTC().Unix()};
    
    return msg, nil
}

func (this *Message) messageUser(server *Server) {
    msg, err := this.formatBody()
    if err != nil {
        return
    }
    
    UID, ok := this.Body["UID"].(string)
    if !ok {
        return
    }
    
    user, err := server.Store.Client(UID)
    if err != nil {
        return
    }
    
    for _, sock := range user {
        sock.buff <- msg
    }
}

func (this *Message) messageAll(server *Server) {
    msg, err := this.formatBody()
    if err != nil {
        return
    }

    clients := server.Store.Clients()
    
    for _, user := range clients {
        for _, sock := range user {
            sock.buff <- msg
        }
    }
    
    return
}

func (this *Message) forwardToRedis(server *Server) {
    msg_str, _ := json.Marshal(this)
    server.Store.redis.Publish("Message", string(msg_str)) //pass the message into redis to send message across cluster    
}


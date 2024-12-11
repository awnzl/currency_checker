See [TASK.md](./TASK.md)

### Build
Run *`make`* to build docker images.  
Run *`make local`* to build everything locally.  

Run *`make proto`* to generate protubuf files locally.  

### Start
Use docker compose to start/down the compose file:  
```
docker-compose up/down
```

Traffic:  
`curl 'http://localhost:8080/?limit=200'`

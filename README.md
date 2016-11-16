# Logepi

### A damn simple logging server.

Logepi comes with a simple approach. Logging should be simple. How?

Simply send your logs over HTTP!

```
POST /log/<table>/ HTTP/1.1
Host: example.com
Content-Type: application/x-www-form-urlencoded
<x1>=<y1>&<x2>=<y2>&<x3>=<y3>
```
Where *table* is the selected logging table, *x* is the column name and *y* is the value to log :thumbsup:

A valid log operation will return a simple -  
`` OK ``  
Where an invalid operation, or an operation that encountered an error will return -  
``ERROR|<Error information>``

###### Server status - 
Currently only a simple "ping" is implemented.  
Simply send a GET to /ping and a pong will be returned.

##### Installation -
tbd

##### Configuration -
A config.yaml with the following information -
```yaml
---
address: 0.0.0.0
port: 8080
database:
  host: localhost
  database: database
  port: 5432
  user: user
  password: password
```

**Notes** -
* The only required column is created_date with the type date
* Currently Logepi is working with a Postgres backend.
* Default address - 0.0.0.0:6080
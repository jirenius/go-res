# Hello World Example

**Tags:** *Models*

## Description
Simple service serving a message to the world.

## Prerequisite

* [Download](https://golang.org/dl/) and install Go
* [Install](https://resgate.io/docs/get-started/installation/) *NATS Server* and *Resgate* (done with 3 docker commands).

## Install and run

```text
git clone https://github.com/jirenius/go-res
cd go-res/examples/01-hello-world
go run .
```

## Things to try out

### Access API through HTTP
* Open the browser and go to:
    ```text
    http://localhost:8080/api/example/model
    ```

### Access API through WebSocket
* Open *Chrome* and go to [resgate.io - resource viewer](https://resgate.io/viewer).
* Type in the resource ID below, and click *View*:
    ```text
    example.model
    ```
    > **Note**
    >
    > Chrome allows websites to connect to *localhost*, while other browsers may give a security error.

### Real time update on static resource
* Stop the project, and change the `"Hello, World!"` message in *main.go*.
* Restart the project and observe how the message is updated in the viewer (see [above](#access-api-through-websocket)).

### Get resource with ResClient
* In the [resgate.io - resource viewer](https://resgate.io/viewer), open the DevTools console (*Ctrl+Shift+J*).
* Type the following command, and press *Enter*:
    ```javascript
    client.get("example.model").then(m => console.log(m.message));
    ```
    > **Note**
    >
    > The resource viewer app stores the *ResClient* instance in the global `client` variable, for easy access.


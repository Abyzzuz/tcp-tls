package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
)

func main() {

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
		return
	}
	fmt.Println("Server is listening on port 8080...")

	for {
		cliente, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		fmt.Println("Client connected:", cliente.RemoteAddr())
		go atenderCliente(cliente)
	}
}

func atenderCliente(cliente net.Conn) {

	//LECTURA DE HANDSHAKE DEL CLIENTE
	buffer := make([]byte, 4096)
	cantidadBytes, err := cliente.Read(buffer)

	if err != nil {
		fmt.Println("Error reading from client:", err)
		return
	}
	serverName, err := extraerSNI(buffer[:cantidadBytes])
	if err != nil {
		fmt.Println("Error extracting SNI:", err)
		cliente.Close()
		return
	}

	fmt.Println("Extracted SNI:", serverName)
	//INYECCION DE SERVER NAME
	destino, err := net.Dial("tcp", serverName+":443")
	if err != nil {
		fmt.Println("Error connecting to destination:", err)
		cliente.Close()
		return
	}
	defer cliente.Close()
	defer destino.Close()

	//SE RECUPERa LO LEIDO
	cantidadBytes, err = destino.Write(buffer[:cantidadBytes])
	if err != nil {
		fmt.Println("Error writing to destination:", err)
		return
	}

	go io.Copy(cliente, destino)
	io.Copy(destino, cliente)
}

func extraerSNI(data []byte) (string, error) {
	var serverName string
	config := &tls.Config{
		GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) {
			serverName = hello.ServerName
			return nil, nil
		},
	}
	clientePipe, serverPipe := net.Pipe()

	go func() {
		clientePipe.Write(data)
		clientePipe.Close()
	}()

	tlsConn := tls.Server(serverPipe, config)
	_ = tlsConn.Handshake()
	// Implement SNI extraction logic here
	return serverName, nil
}

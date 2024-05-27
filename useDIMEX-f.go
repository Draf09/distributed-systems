// Construido como parte da disciplina: Sistemas Distribuidos - PUCRS - Escola Politecnica
//  Professor: Fernando Dotti  (https://fldotti.github.io/)
// Uso p exemplo:
//   go run useDIMEX-f.go 0 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
//   go run useDIMEX-f.go 1 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
//   go run useDIMEX-f.go 2 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
// ----------
// LANCAR N PROCESSOS EM SHELL's DIFERENTES, UMA PARA CADA PROCESSO.
// para cada processo fornecer: seu id único (0, 1, 2 ...) e a mesma lista de processos.
// o endereco de cada processo é o dado na lista, na posicao do seu id.
// no exemplo acima o processo com id=1  usa a porta 6001 para receber e as portas
// 5000 e 7002 para mandar mensagens respectivamente para processos com id=0 e 2
// -----------
// Esta versão supõe que todos processos tem acesso a um mesmo arquivo chamado "mxOUT.txt"
// Todos processos escrevem neste arquivo, usando o protocolo dimex para exclusao mutua.
// Os processos escrevem "|." cada vez que acessam o arquivo.   Assim, o arquivo com conteúdo
// correto deverá ser uma sequencia de
// |.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.
// |.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.
// |.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.|.
// etc etc ...     ....  até o usuário interromper os processos (ctl c).
// Qualquer padrao diferente disso, revela um erro.
//      |.|.|.|.|.||..|.|.|.  etc etc  por exemplo.
// Se voce retirar o protocolo dimex vai ver que o arquivo poderá entrelacar
// "|."  dos processos de diversas diferentes formas.
// Ou seja, o padrão correto acima é garantido pelo dimex.
// Ainda assim, isto é apenas um teste.  E testes são frágeis em sistemas distribuídos.

// useDIMEX-f.go
package main

import (
	"distributed-systems/SNAPSHOT"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

func startListener(port int) {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error starting listener on port %d: %v\n", port, err)
		os.Exit(1)
	}
	defer listener.Close()
	fmt.Printf("Listening on port %d\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	// Handle the connection
}

func main() {

	if len(os.Args) < 2 {
		fmt.Println("Please specify at least one address:port!")
		fmt.Println("go run usaDIMEX-f.go 0 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
		fmt.Println("go run usaDIMEX-f.go 1 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
		fmt.Println("go run usaDIMEX-f.go 2 127.0.0.1:5000  127.0.0.1:6001  127.0.0.1:7002 ")
		return
	}

	id, _ := strconv.Atoi(os.Args[1])
	addresses := os.Args[2:]
	// fmt.Print("id: ", id, "   ") fmt.Println(addresses)

	if id == 0 {
		go startListener(5000)
	} else if id == 1 {
		go startListener(6001)
	} else if id == 2 {
		go startListener(7002)
	}
	time.Sleep(5 * time.Second) // Give the listener time to start

	dmx := SNAPSHOT.NewDIMEX(addresses, id, true)
	fmt.Println(dmx)

	// abre arquivo que TODOS processos devem/poder usar
	file, err := os.OpenFile("./mxOUT.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close() // Ensure the file is closed at the end of the function

	for {
		fmt.Println("[ APP id: ", id, " PEDE   MX ]")
		dmx.Req <- SNAPSHOT.ENTER
		//fmt.Println("[ APP id: ", id, " ESPERA MX ]")
		// ESPERA LIBERACAO DO MODULO DIMEX
		<-dmx.Ind //

		// A PARTIR DAQUI ESTA ACESSANDO O ARQUIVO SOZINHO
		_, err = file.WriteString("|") // marca entrada no arquivo
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}
		file.Sync() // garante que a escrita foi feita no disco

		fmt.Println("[ APP id: ", id, " *EM*   MX ]")

		_, err = file.WriteString(".") // marca saida no arquivo
		if err != nil {
			fmt.Println("Error writing to file:", err)
			return
		}

		// Initiate a snapshot after accessing the critical section
		fmt.Printf("[ APP id: %d initiated snapshot ]\n", id)
		dmx.InitiateSnapshot()

		// AGORA VAI LIBERAR O ARQUIVO PARA OUTROS
		dmx.Req <- SNAPSHOT.EXIT //
		fmt.Println("[ APP id: ", id, " FORA   MX ]")
	}
}

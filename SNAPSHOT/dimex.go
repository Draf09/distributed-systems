/*
	  Construido como parte da disciplina: FPPD - PUCRS - Escola Politecnica
	    Professor: Fernando Dotti  (https://fldotti.github.io/)
	    Modulo representando Algoritmo de Exclusão Mútua Distribuída:
	    Semestre 2023/1
		Aspectos a observar:
		   mapeamento de módulo para estrutura
		   inicializacao
		   semantica de concorrência: cada evento é atômico
		   							  módulo trata 1 por vez
		Q U E S T A O
		   Além de obviamente entender a estrutura ...
		   Implementar o núcleo do algoritmo ja descrito, ou seja, o corpo das
		   funcoes reativas a cada entrada possível:
		   			handleUponReqEntry()  // recebe do nivel de cima (app)
					handleUponReqExit()   // recebe do nivel de cima (app)
					handleUponDeliverRespOk(msgOutro)   // recebe do nivel de baixo
					handleUponDeliverReqEntry(msgOutro) // recebe do nivel de baixo
*/
package SNAPSHOT

import (
	PP2PLink "distributed-systems/PP2PLink"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// // Define the port mapping
// var portMapping = map[int]int{
// 	0: 5000,
// 	1: 6001,
// 	2: 7002,
// 	// Add more mappings as needed
// }

// ------------------------------------------------------------------------------------
// ------- principais tipos
// ------------------------------------------------------------------------------------

type MarkerMessage struct {
	SenderID   int
	SnapshotID int
}

type State int // enumeracao dos estados possiveis de um processo

const (
	noMX   State = iota
	wantMX       // quer acessar
	inMX
)

type dmxReq int // enumeracao dos estados possiveis de um processo , dmx = DIMEX - pedido de acesso
const (
	ENTER dmxReq = iota
	EXIT
)

type dmxResp struct { // mensagem do módulo DIMEX infrmando que pode acessar - pode ser somente um sinal (vazio)
	// mensagem para aplicacao indicando que pode prosseguir

}

type DIMEX_Module struct {
	Req          chan dmxReq  // canal para receber pedidos da aplicacao (REQ e EXIT)
	Ind          chan dmxResp // canal para informar aplicacao que pode acessar
	SnapshotReq  chan MarkerMessage
	SnapshotResp chan MarkerMessage
	addresses    []string // endereco de todos, na mesma ordem
	id           int      // identificador do processo - é o indice no array de enderecos acima
	st           State    // estado deste processo na exclusao mutua distribuida
	waiting      []bool   // processos aguardando tem flag true
	lcl          int      // relogio logico local
	reqTs        int      // timestamp local da ultima requisicao deste processo
	nbrResps     int      // numero de respostas recebidas
	dbg          bool
	Snapshot     map[int]string // map of snapshots

	Pp2plink *PP2PLink.PP2PLink // acesso aa comunicacao enviar por PP2PLinq.Req  e receber por PP2PLinq.Ind
}

// ------------------------------------------------------------------------------------
// ------- inicializacao
// ------------------------------------------------------------------------------------

func NewDIMEX(_addresses []string, _id int, _dbg bool) *DIMEX_Module {

	p2p := PP2PLink.NewPP2PLink(_addresses[_id], _dbg)

	dmx := &DIMEX_Module{
		Req: make(chan dmxReq, 1),
		Ind: make(chan dmxResp, 1),

		addresses:    _addresses,
		id:           _id,
		st:           noMX, //st = estado do processo
		waiting:      make([]bool, len(_addresses)),
		lcl:          0,
		reqTs:        0,
		dbg:          _dbg,
		Snapshot:     make(map[int]string),
		Pp2plink:     p2p,
		SnapshotReq:  make(chan MarkerMessage),
		SnapshotResp: make(chan MarkerMessage),
	}

	for i := 0; i < len(dmx.waiting); i++ {
		dmx.waiting[i] = false
	}
	dmx.Start()
	dmx.outDbg("Init DIMEX!")
	return dmx
}

// ------------------------------------------------------------------------------------
// ------- nucleo do funcionamento
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) Start() {
	go func() {
		for {
			select {
			case msg := <-module.Pp2plink.Ind:
				if strings.Contains(msg.Message, "snapshot") {
					module.handleSnapshotMessage(msg)
				} else if strings.Contains(msg.Message, "marker") {
					// Parse the marker message
					parts := strings.Split(msg.Message, ",")
					senderID, _ := strconv.Atoi(parts[0])
					snapshotID, _ := strconv.Atoi(parts[1])
					marker := MarkerMessage{SenderID: senderID, SnapshotID: snapshotID}
					module.handleMarkerMessage(marker)
				} else if strings.Contains(msg.Message, "respOK") {
					module.outDbg("         <<<---- responde! " + msg.Message)
					module.handleUponDeliverRespOk(msg)
				} else if strings.Contains(msg.Message, "reqEntry") {
					module.outDbg("          <<<---- pede??  " + msg.Message)
					module.handleUponDeliverReqEntry(msg)
				}
			case dmxR := <-module.Req:
				if dmxR == ENTER {
					module.outDbg("app pede mx")
					module.handleUponReqEntry()
				} else if dmxR == EXIT {
					module.outDbg("app libera mx")
					module.handleUponReqExit()
				}
			case marker := <-module.SnapshotResp:
				module.handleMarkerMessage(marker)
				fmt.Printf("[Snapshot] Process %d received marker from Process %d\n", module.id, marker.SenderID)
			}
		}
	}()
}

// ------------------------------------------------------------------------------------
// ------- tratamento de pedidos vindos da aplicacao
// ------- UPON ENTRY
// ------- UPON EXIT
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) handleUponReqEntry() {
	/*
					upon event [ dmx, Entry  |  r ]  do
		    			lts.ts++
		    			myTs := lts
		    			resps := 0
		    			para todo processo p
							trigger [ pl , Send | [ reqEntry, r, myTs ]
		    			estado := queroSC
	*/
	println("===== handleUponReqEntry =====", module.id)
	module.lcl++              // lcl = logical clock
	module.reqTs = module.lcl // atribui o valor do relogio local ao timestamp da requisicao

	module.TakeSnapshot()

	var p1 string
	var leadingSpc string // cria uma string com espaços em branco
	for i := 0; i < len(module.addresses); i++ {
		if i != module.id { // verifica  nao envia para si mesmo
			module.sendToLink(module.addresses[i], "reqEntry", "   ")
		}
		p1 = module.addresses[i] // cria um processo
		message := module.stringify("reqEntry", module.id, module.lcl)
		leadingSpc = strings.Repeat("  ", len(p1)-len(module.addresses[module.id])+1)
		module.sendToLink(p1, message, leadingSpc)

	}
	module.st = wantMX // st = state
}

func (module *DIMEX_Module) handleUponReqExit() {
	/*
						upon event [ dmx, Exit  |  r  ]  do
		       				para todo [p, r, ts ] em waiting
		          				trigger [ pl, Send | p , [ respOk, r ]  ]
		    				estado := naoQueroSC
							waiting := {}
	*/
	println("===== handleUponReqExit =====", module.id)
	for i := 0; i < len(module.waiting); i++ {
		if module.waiting[i] {
			p1 := module.addresses[i]
			message := module.stringify("respOk", module.id, module.lcl)
			leadingSpc := strings.Repeat(" ", 22-len(message))
			module.sendToLink(p1, message, leadingSpc)
			module.waiting[i] = false
		}
	}
	module.st = noMX
}

// ------------------------------------------------------------------------------------
// ------- tratamento de mensagens de outros processos
// ------- UPON respOK
// ------- UPON reqEntry
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) handleUponDeliverRespOk(msgOutro PP2PLink.PP2PLink_Ind_Message) {
	/*
						upon event [ pl, Deliver | p, [ respOk, r ] ]
		      				resps++
		      				se resps = N
		    				então trigger [ dmx, Deliver | free2Access ]
		  					    estado := estouNaSC

	*/
	fmt.Println("====== handleUponDeliverRespOk =======", module.id)
	module.nbrResps++
	N := len(module.addresses) - 1
	fmt.Println("nbrResps: ", module.nbrResps, "N: ", N)
	if module.nbrResps == N {
		module.Ind <- dmxResp{}
		module.st = inMX
	}
}

func (module *DIMEX_Module) handleUponDeliverReqEntry(msgOutro PP2PLink.PP2PLink_Ind_Message) {
	// outro processo quer entrar na SC
	/*
						upon event [ pl, Deliver | p, [ reqEntry, r, rts ]  do
		     				se (estado == naoQueroSC)   OR
		        				 (estado == QueroSC AND  myTs >  ts)
							então  trigger [ pl, Send | p , [ respOk, r ]  ]
		 					senão
		        				se (estado == estouNaSC) OR
		           					 (estado == QueroSC AND  myTs < ts)
		        				então  postergados := postergados + [p, r ]
		     					lts.ts := max(lts.ts, rts.ts)
	*/
	fmt.Println("====== handleUponDeliverReqEntry =======", module.id)
	module.lcl++
	myTs := module.lcl
	resps := 0
	//module.InitiateSnapshot()

	for _, p := range module.addresses { //itera sobre todos os processos e envia reqEntry para todos
		pID, _ := strconv.Atoi(p)
		if pID != module.id {
			go func(peerID int, myTs int) { // go routine para enviar a mensagem
				message := fmt.Sprintf("[pl, Send | %d, [reqEntry, %d, %d]]", module.id, module.id, myTs)
				module.sendToLink(p, message, "") // envia a mensagem para o processo
			}(pID, myTs) // envia a mensagem para o processo
		}
	}

	module.st = wantMX
	module.reqTs = myTs // atribui o valor do relogio local ao timestamp da requisicao

	fmt.Printf("[DEBUG] Process %d: Sent requests for timestamp %d\n", module.id, myTs)

	for resps < module.nbrResps { // enquanto o numero de respostas for menor que o numero de respostas esperadas
		msgOutro := <-module.Pp2plink.Ind                 // recebe a mensagem do outro processo
		if strings.Contains(msgOutro.Message, "respOk") { // se a mensagem contiver "respOk"
			module.outDbg("         <<<---- responde! " + msgOutro.Message)
			module.handleUponDeliverRespOk(msgOutro) // ENTRADA DO ALGORITMO - responde
			resps++                                  // incrementa o numero de respostas
		}
	}

	fmt.Printf("[DEBUG] Process %d: Received all %d responses\n", module.id, module.nbrResps)

	// Um atraso foi criado para simular o tempo de escrita no arquivo
	// time.Sleep(500 * time.Millisecond)

	// Usa o canal para indicar que terminou a região crítica
	module.Ind <- dmxResp{}
	module.st = inMX
}

// ------------------------------------------------------------------------------------
// ------- funcoes de ajuda
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) sendToLink(address string, content string, space string) {
	module.outDbg(space + " ---->>>>   to: " + address + "     msg: " + content)
	module.Pp2plink.Req <- PP2PLink.PP2PLink_Req_Message{
		To:      address,
		Message: content}
}

func before(oneId, oneTs, othId, othTs int) bool {
	if oneTs < othTs {
		return true
	} else if oneTs > othTs {
		return false
	} else {
		return oneId < othId
	}
}

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func stringify(_mensagem string, _id int, _relogioLocal int) string {
	id := strconv.Itoa(_id)
	relogioLocal := strconv.Itoa(_relogioLocal)
	return fmt.Sprintf("(%s) %s ts=%s", id, _mensagem, relogioLocal)
}

func (module *DIMEX_Module) parseMsg(msg string) (mensagem string, id int, lcl int) {
	// returns ("id") string
	id_full := getWord(msg, 0)
	_id := remove(remove(id_full, "("), ")")
	id, _ = strconv.Atoi(_id)

	// returns text content (respOk, reqSent, etc)
	msg = getWord(msg, 0)
	fmt.Println("msg: ", msg)

	// Returns ts="ts"
	lcl_full := getWord(msg, 2)
	// Returns "ts" string
	_relogioLocal := remove(lcl_full, "ts=")
	// Returns ts int
	lcl, _ = strconv.Atoi(_relogioLocal)
	return mensagem, id, lcl

}

func (module *DIMEX_Module) stringify(_mensagem string, _id int, _relogioLocal int) string {
	id := strconv.Itoa(_id)
	relogioLocal := strconv.Itoa(_relogioLocal)
	return fmt.Sprintf("(%s) %s ts=%s", id, _mensagem, relogioLocal)
}

func getWord(str string, index int) string {
	return strings.Split(str, " ")[index]
}

func remove(str, old string) string {
	return strings.Replace(string(str), old, "", -1)
}

func (module *DIMEX_Module) outDbg(s string) {
	if module.dbg {
		fmt.Println(". . . . . . . . . . . . [ DIMEX : " + s + " ]")
	}
}

func (module *DIMEX_Module) RecordState() string {
	// convert state to string
	stateStr := strconv.Itoa(int(module.st))
	lclStr := strconv.Itoa(module.lcl)

	// combine state and lgical clock
	recordedState := "State: " + stateStr + "Logical Clock: " + lclStr
	return recordedState
}

func (module *DIMEX_Module) handleSnapshotMessage(msg PP2PLink.PP2PLink_Ind_Message) {
	// Parse the snapshot message
	parts := strings.Split(msg.Message, ",")
	if len(parts) < 3 {
		fmt.Printf("Invalid snapshot message received: %s\n", msg.Message)
		return
	}

	fromId, err := strconv.Atoi(parts[1])
	if err != nil {
		fmt.Printf("Error converting from ID to integer: %v\n", err)
		return
	}

	snapshot := strings.Join(parts[2:], ",")
	fmt.Printf("[Snapshot] Process %d received snapshot from Process %d: %s\n", module.id, fromId, snapshot)

	module.Snapshot[fromId] = snapshot
}

func (module *DIMEX_Module) TakeSnapshot() {
	// Record local state
	module.Snapshot[module.id] = module.RecordState()
	fmt.Printf("[Snapshot] Process %d recorded local state: %s\n", module.id, module.Snapshot[module.id])
	// Send marker to all other processes
	for _, address := range module.addresses {
		if address != module.Pp2plink.Address {
			markerMessage := fmt.Sprintf("%d,%d", module.id, module.lcl)
			module.sendToLink(address, "marker"+markerMessage, "")
			fmt.Printf("[Snapshot] Process %d sent marker to %s\n", module.id, address)
		}
	}
}

func (module *DIMEX_Module) InitiateSnapshot() {
	// Record local state of current process
	module.Snapshot[module.id] = module.RecordState()
	fmt.Printf("[Snapshot] Process %d recorded local state: %s\n", module.id, module.Snapshot[module.id])

	// Send marker to all other processes
	for _, address := range module.addresses {
		if address != module.Pp2plink.Address {
			markerMessage := fmt.Sprintf("marker,%d,%d", module.id, module.lcl)
			module.sendToLink(address, markerMessage, "")
			fmt.Printf("[Snapshot] Process %d sent marker to %s\n", module.id, address)
		}
	}

	// Wait for the snapshot markers from other processes
	for i := 0; i < len(module.addresses)-1; i++ {
		marker := <-module.SnapshotResp
		fmt.Printf("[Snapshot] Process %d received marker from Process %d\n", module.id, marker.SenderID)
	}

	// Send local snapshot to the aggregator process (assuming it's the process with ID 0)
	if module.id != 0 {
		snapshotMessage := fmt.Sprintf("snapshot,%d,%s", module.id, module.Snapshot[module.id])
		module.sendToLink(module.addresses[0], snapshotMessage, "")
		fmt.Printf("[Snapshot] Process %d sent snapshot to aggregator (Process 0)\n", module.id)
	} else {
		fmt.Println("[Snapshot] Aggregator process (Process 0) waiting for snapshots")
	}

	// Aggregate the snapshots (only for the aggregator process)
	if module.id == 0 {
		module.AggregateSnapshots()
	}
}

func (module *DIMEX_Module) handleMarkerMessage(marker MarkerMessage) {
	fmt.Printf("[Snapshot] Process %d received marker from Process %d\n", module.id, marker.SenderID)

	if _, exists := module.Snapshot[module.id]; !exists {
		module.TakeSnapshot()
	}

	// Send marker to all other processes
	for _, address := range module.addresses {
		if address != module.Pp2plink.Address {
			markerMessage := fmt.Sprintf("%d,%d", module.id, marker.SnapshotID)
			module.sendToLink(address, markerMessage, "")
			fmt.Printf("[Snapshot] Process %d sent marker to %s\n", module.id, address)
		}
	}

	// Send local snapshot to the aggregator process (assuming it's the process with ID 0)
	if module.id != 0 {
		snapshotMessage := fmt.Sprintf("snapshot,%d,%s", module.id, module.Snapshot[module.id])
		module.sendToLink(module.addresses[0], snapshotMessage, "")
		fmt.Printf("[Snapshot] Process %d sent snapshot to aggregator (Process 0)\n", module.id)
	} else {
		fmt.Println("[Snapshot] Aggregator process (Process 0) waiting for snapshots")
	}
}

func (module *DIMEX_Module) receiveSnapshotFromProcess(processID int) string {
	fmt.Printf("[Snapshot] Process %d receiving snapshot from Process %d\n", module.id, processID)

	// Retrieve the port from the addresses map (assuming it's set up correctly)
	port, err := strconv.Atoi(strings.Split(module.addresses[processID], ":")[1])
	if err != nil {
		fmt.Printf("Invalid port for process %d: %v\n", processID, err)
		return ""
	}

	// Create a TCP connection to the specified process with retry mechanism
	addr := fmt.Sprintf("%s:%d", strings.Split(module.addresses[processID], ":")[0], port)
	var conn net.Conn
	for i := 0; i < 3; i++ { // Retry up to 3 times
		conn, err = net.Dial("tcp", addr)
		if err == nil {
			break
		}
		fmt.Printf("Error connecting to process %d: %v (attempt %d)\n", processID, err, i+1)
		time.Sleep(1 * time.Second) // Wait before retrying
	}
	if err != nil {
		fmt.Printf("Failed to connect to process %d after 3 attempts: %v\n", processID, err)
		return ""
	}
	defer conn.Close()

	// Receive the snapshot data from the process
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Printf("Error reading snapshot from process %d: %v\n", processID, err)
		return ""
	}

	snapshotData := string(buf[:n])
	parts := strings.Split(snapshotData, ",")
	if len(parts) == 2 {
		receivedProcessID, _ := strconv.Atoi(parts[0])
		snapshot := parts[1]
		module.Snapshot[receivedProcessID] = snapshot
		fmt.Printf("[Snapshot] Process %d received snapshot from Process %d: %s\n", module.id, receivedProcessID, snapshot)
		return snapshot
	} else {
		fmt.Printf("Invalid snapshot data received from Process %d\n", processID)
		return ""
	}
}

func (module *DIMEX_Module) AggregateSnapshots() {
	// Create a map to store snapshots from all processes
	snapshots := make(map[int]string)

	// Collect snapshots from all processes
	for i := 0; i < len(module.addresses); i++ {
		if i != module.id {
			// Receive snapshot from the process
			snapshot := module.receiveSnapshotFromProcess(i)
			if snapshot != "" {
				snapshots[i] = snapshot
			} else {
				fmt.Printf("[Snapshot] Failed to receive snapshot from Process %d\n", i)
			}
		} else {
			snapshots[module.id] = module.Snapshot[module.id]
			fmt.Printf("[Snapshot] Process %d added its own snapshot: %s\n", module.id, module.Snapshot[module.id])
		}
	}

	// Combine snapshots into a global snapshot
	var globalSnapshot strings.Builder
	globalSnapshot.WriteString("Global Snapshot:\n")
	for processID, snapshot := range snapshots {
		globalSnapshot.WriteString(fmt.Sprintf("Process %d: %s\n", processID, snapshot))
	}

	// Write the global snapshot to mxOUT.txt
	file, err := os.OpenFile("./mxOUT.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	_, err = file.WriteString(globalSnapshot.String())
	if err != nil {
		fmt.Printf("Error writing to file: %v\n", err)
		return
	}

	fmt.Println("[Snapshot] Global snapshot written to mxOUT.txt")
	fmt.Println("[Snapshot] Global snapshot:", globalSnapshot.String())

}

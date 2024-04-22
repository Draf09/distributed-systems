/*  Construido como parte da disciplina: FPPD - PUCRS - Escola Politecnica
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

package DIMEX

import (
	PP2PLink "distributed-systems/PP2PLink"
	"fmt"
	"strconv"
	"strings"
)

// ------------------------------------------------------------------------------------
// ------- principais tipos
// ------------------------------------------------------------------------------------

type State int // enumeracao dos estados possiveis de um processo

const (
	noMX State = iota
	wantMX // quer acessar
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
	Req       chan dmxReq  // canal para receber pedidos da aplicacao (REQ e EXIT)
	Ind       chan dmxResp // canal para informar aplicacao que pode acessar
	addresses []string     // endereco de todos, na mesma ordem
	id        int          // identificador do processo - é o indice no array de enderecos acima
	st        State        // estado deste processo na exclusao mutua distribuida
	waiting   []bool       // processos aguardando tem flag true
	lcl       int          // relogio logico local
	reqTs     int          // timestamp local da ultima requisicao deste processo
	nbrResps  int		// numero de respostas recebidas
	dbg       bool
	localState State	// estado local do processo
	incomingState State	// estado recebido de outro processo
	Pp2plink *PP2PLink.PP2PLink // acesso aa comunicacao enviar por PP2PLinq.Req  e receber por PP2PLinq.Ind
	snapShotInProgress bool	// flag para indicar se o snapshot esta em progresso
	stateRecorded bool	// flag para indicar se o estado foi gravado
}

// snapshot message
type SnapshotMessage struct {
	SenderID int
	SnapshotID int
}


// ------------------------------------------------------------------------------------
// ------- inicializacao
// ------------------------------------------------------------------------------------

func NewDIMEX(_addresses []string, _id int, _dbg bool) *DIMEX_Module {

	p2p := PP2PLink.NewPP2PLink(_addresses[_id], _dbg)

	dmx := &DIMEX_Module{
		Req: make(chan dmxReq, 1),
		Ind: make(chan dmxResp, 1),

		addresses: _addresses,
		id:        _id,
		st:        noMX, //st = estado do processo
		waiting:   make([]bool, len(_addresses)),
		lcl:       0,
		reqTs:     0,
		dbg:       _dbg,

		Pp2plink: p2p}

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
			case dmxR := <-module.Req: // vindo da  aplicação
				if dmxR == ENTER {
					module.outDbg("app pede mx")
					module.handleUponReqEntry() // ENTRADA DO ALGORITMO

				} else if dmxR == EXIT {
					module.outDbg("app libera mx")
					module.handleUponReqExit() // ENTRADA DO ALGORITMO
				}

			case msgOutro := <-module.Pp2plink.Ind: // vindo de outro processo
				//fmt.Printf("dimex recebe da rede: ", msgOutro)
				if strings.Contains(msgOutro.Message, "respOK") {
					module.outDbg("         <<<---- responde! " + msgOutro.Message)
					module.handleUponDeliverRespOk(msgOutro) // ENTRADA DO ALGORITMO

				} else if strings.Contains(msgOutro.Message, "reqEntry") {
					module.outDbg("          <<<---- pede??  " + msgOutro.Message)
					module.handleUponDeliverReqEntry(msgOutro) // ENTRADA DO ALGORITMO

				}
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
	module.lcl++  // lcl = logical clock
	module.reqTs = module.lcl // atribui o valor do relogio local ao timestamp da requisicao
	module.nbrResps = 0
	var p1 string
	var leadingSpc string // cria uma string com espaços em branco
	for i := 0; i < len(module.addresses); i++ {
		if(i != module.id) { // verifica  nao envia para si mesmo
			module.sendToLink(module.addresses[i], "reqEntry", "   ")
		}
		p1 = module.addresses[i] // cria um processo
		message := module.stringify("reqEntry", module.id, module.lcl)
		leadingSpc = strings.Repeat("  ", len(p1) - len(module.addresses[module.id]) + 1)
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

	// check if is a snapshot message request
	if strings.Contains(msgOutro.Message, "snapshotRequest") {
		snapshot, err := module.parseSnapshotMessage(msgOutro.Message)
		if err != nil {
			fmt.Println("Error parsing snapshot message:", err)
			return
		}
		// Handle snapshot request
		module.handleSnapshotRequest(snapshot)
		return
	}

	// handle regular request entry 
	for _, p := range module.addresses {
		pID, _ := strconv.Atoi(p) 
		if pID != module.id {
			go func(peerID int, myTs int) {
				message := fmt.Sprintf("[pl | Send | %d, [reqEntry, %d, %d]]", module.id, module.id, myTs)
				module.sendToLink(p, message, "   ")
			}(pID, myTs)
		}
	}
	module.st = wantMX
	module.reqTs = myTs

	fmt.Println("[DEBUG] Process %d: Sent request for timestamp %d\n", module.id, myTs)
	
	for resps < module.nbrResps {
		msgOutro := <-module.Pp2plink.Ind
		if strings.Contains(msgOutro.Message, "respOk") {
			module.outDbg("         <<<---- responde! " + msgOutro.Message)
			module.handleUponDeliverRespOk(msgOutro) // ENTRADA DO ALGORITMO
			resps++
		}
	}

	fmt.Println("[DEBUG] Process %d: Received all responses\n", module.id, module.nbrResps)
	
	module.Ind <- dmxResp{}
	module.st = inMX
}

		
func (module *DIMEX_Module) handleSnapshotRequest(snapshot SnapshotMessage) {
	module.recordState()

	for _, address := range module.addresses {
		if address != module.Pp2plink.Address {
			module.sendToLink(address, "snapshotState", module.SerializeSnapshotMessage(SnapshotMessage{SenderID: snapshot.SenderID, SnapshotID: snapshot.SnapshotID}))
		}
	}
}

// ------------------------------------------------------------------------------------
// ------- funcoes de ajuda
// ------------------------------------------------------------------------------------

func (module *DIMEX_Module) sendToLink(address string, content string, space string) {
	if module.snapShotInProgress {
		// append the snapshot token to the message content , the Id 
		contentWithToken := fmt.Sprintf("%s snapshotToken=%d", content, module.snapShotInProgress)
		module.outDbg(space + " ---->>>>   to: " + address + "     msg: " + contentWithToken)
		module.Pp2plink.Req <- PP2PLink.PP2PLink_Req_Message{
			To:      address,
			Message: contentWithToken,
		}
	} else {
		module.outDbg(space + " ---->>>>   to: " + address + "     msg: " + content)
		module.Pp2plink.Req <- PP2PLink.PP2PLink_Req_Message{
			To:      address,
			Message: content,
		}
	}
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
	id , _ = strconv.Atoi(_id)

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

//////////////////////// snapshot Chandy-Lamport //////////////////////////
/*
implementacao do algoritmo de snapshot Chandy-Lamport
https://en.wikipedia.org/wiki/Chandy%E2%80%93Lamport_algorithm
https://www.geeksforgeeks.org/chandy-lamport-algorithm-for-distributed-snapshot/

1. processo p0 manda uma mensagem para si mesmo com "take snapshot"
2. seja p1 o processo do qual p1 recebe a mensagem "take snapshot" pela primeira vez
    Ao receber p_i, grava seu estado local "sigma" e envia "marker" para todos os canais em OUT_i(todos os canais FIFO)
O estado de x_fi é setado vazio, P_i inicia a gravacao de mensagens recebidas de cada um de seus outros canais em IN_i

3. Seja o processo no qual p_j recebe a mensagem "take snapshot" depois da primeira vez, p_i para de gravar mensagens de p_s e declara o estado x_s,i como sendo as mensagens gravadas.

Quando o processo p_i tiver recebido "take snapshot" em todos os canais, sua contribuição termina.
Este termina, pois a mensagem é enviada somente uma vez em cada canal de saída.
*/

func (module *DIMEX_Module) ChandyLamport() {
	// 1. processo p0 manda uma mensagem para si mesmo com "take snapshot"
	module.sendToLink(module.addresses[module.id], "take snapshot", "   ")
	// 2. seja p1 o processo do qual p1 recebe a mensagem "take snapshot" pela primeira vez
	// Ao receber p_i, grava seu estado local "sigma" e envia "marker" para todos os canais em OUT_i
	// O estado de x_fi é setado vazio, P_i inicia a gravacao de mensagens recebidas de cada um de seus outros canais em IN_i
	sigma := module.stringify("sigma", module.id, module.lcl)
	for i := 0; i < len(module.addresses); i++ {
		if i != module.id {
			module.sendToLink(module.addresses[i], "marker", "   ")
		}
	}
	x := make([]string, len(module.addresses))
	for i := 0; i < len(module.addresses); i++ {
		x[i] = ""
	}

	// 3. Seja o processo no qual p_j recebe a mensagem "take snapshot" depois da primeira vez, p_i para de gravar mensagens de p_s e declara o estado x_s,i como sendo as mensagens gravadas.
	for i := 0; i < len(module.addresses); i++ {
		if i != module.id {
			msgOutro := <-module.Pp2plink.Ind
			if strings.Contains(msgOutro.Message, "take snapshot") {
				x[i] = sigma
			}
		}
	}

	// Quando o processo p_i tiver recebido "take snapshot" em todos os canais, sua contribuição termina

	// Este termina, pois a mensagem é enviada somente uma vez em cada canal de saída.
	// end

}

func (module *DIMEX_Module) InitiateSnapshot() {
    // Save local state
    module.recordState()

    // Send snapshot request to all other processes
    for _, address := range module.addresses {
        if address != module.Pp2plink.Address {
            module.sendToLink(address, "snapshotRequest", module.SerializeSnapshotMessage(SnapshotMessage{SenderID: module.id, SnapshotID: module.lcl}))
        }
    }
}

func (module *DIMEX_Module) SerializeSnapshotMessage(snapshot SnapshotMessage) string {
	return fmt.Sprintf("snapshot:%d:%d", snapshot.SenderID, snapshot.SnapshotID)
}

func (module *DIMEX_Module) handleUponSnapshot(msgOutro PP2PLink.PP2PLink_Ind_Message) {
	// record local state
	module.recordState()
	snapshot, err := module.parseSnapshotMessage(msgOutro.Message)
	if err != nil {
		fmt.Println("Error parsing snapshot message:", err)
		return
	}
	// Record the local state if not already state 
	if !module.stateRecorded {
		module.recordState()
	}
	// Record the channel state 
	module.recordState()
	// Send snapshot message to all other processes
	fmt.Println("Process ", module.id, " received snapshot message from process ", snapshot.SnapshotID)
	for _, address := range module.addresses {
		if address != module.Pp2plink.Address {
			module.sendToLink(address, "snapshotState", module.SerializeSnapshotMessage(SnapshotMessage{SenderID: snapshot.SenderID, SnapshotID: snapshot.SnapshotID}))
		}
	}
}

func (module *DIMEX_Module) parseSnapshotMessage(msg string) (SnapshotMessage, error) {
    // Assuming the message is serialized in a specific format, e.g., "snapshot:senderID:snapshotID"
    parts := strings.Split(msg, ":")
    if len(parts) != 3 || parts[0] != "snapshot" {
        return SnapshotMessage{}, fmt.Errorf("invalid snapshot message format")
    }
    senderID, err := strconv.Atoi(parts[1])
    if err != nil {
        return SnapshotMessage{}, fmt.Errorf("invalid senderID in snapshot message: %v", err)
    }
    snapshotID, err := strconv.Atoi(parts[2])
    if err != nil {
        return SnapshotMessage{}, fmt.Errorf("invalid snapshotID in snapshot message: %v", err)
    }
    return SnapshotMessage{SenderID: senderID, SnapshotID: snapshotID}, nil
}


func (module *DIMEX_Module) sendMarker(address string) {
	marker := createMarkerMessage(module.id)
	module.sendToLink(address, "marker", marker)
}

func createMarkerMessage(senderID int) string {
	return fmt.Sprintf("marker:%d", senderID)
}

func (module *DIMEX_Module) handleMarkerMessage(msgOutro PP2PLink.PP2PLink_Ind_Message) {
	senderID, err := parseMarkerMessage(msgOutro.Message)
	if err != nil {
		fmt.Println("Error parsing marker message:", err)
		return
	}
	// Record the local state if not already state 
	if !module.stateRecorded {
		module.recordState()
	}
	// Record the channel state 
	module.recordState()
	// Send marker message to all other processes
	fmt.Println("Process ", module.id, " received marker message from process ", senderID)
	for _, address := range module.addresses {
		if address != module.Pp2plink.Address {
			module.sendToLink(address, "marker", createMarkerMessage(senderID))
		}
	}
}

func parseMarkerMessage(msg string) (int, error) {
    parts := strings.Split(msg, ":")
    if len(parts) != 2 || parts[0] != "marker" {
	return 0, fmt.Errorf("invalid marker message format")
    }
    senderID, err := strconv.Atoi(parts[1])
    if err != nil {
	return 0, fmt.Errorf("invalid senderID in marker message: %v", err)
    }
    return senderID, nil
}

// Record the local state
func (module *DIMEX_Module) recordState() {
	module.localState = module.st
	for i := 0; i < len(module.waiting); i++ {
		module.incomingState = module.st
	}
	// Record the local state of the process
	module.stateRecorded = true

	fmt.Println("Recorded local state:", module.localState)
}


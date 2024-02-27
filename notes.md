# Notes for Distributed Systems Course 2024/1

Aula de Introdução  
   Nesta Semana: Leitura dirigida do artigo. 
   "What are good models and what models are good." da pasta artigos no módulo Materiais.
   Com base no texto, procure responder às seguintes questões:
1. O autor critica o ensino de computação dizendo, em alguma parte, que: 

...
The study builds intuition for a particular model of computation—one that involves a single sequential process and a uniform access-time memory. Unfortunately, this is neither an accurate nor useful model for the systems that concern us. ...

Você concorda com esta afirmação (no seu contexto)?  Justifique.

Sim, concordo. O que o autor se refere, problemas relacionados à indedecibilidade e classes de complexidade, que são ensinados nas disciplinas de ensino de algoritmos com enfoque a apenas um modelo computacional. Quando se trata de mais de um modelo, é necessário ter uma abordagem diferente pois o comportamento destes algoritmos e modelos em conjunto diferem bastante, se forem analisados de forma distribuída, ou envolve a criação de outros modelos, pelo fato de terem características bem diferentes, como comunicação entre processos, sujeitos à falhas, falhas de rede ocasionando timeouts, entre outros. 


2. Na seção "A Coordination Problem", que estratégias de raciocínio o autor utilizou para provar que o
problema não tem solução ?

O racionío utilizado foi fazer uma prova, ou uma derivação por contradição, pois se as premissas de uma suposta implementação de um protocolo forem:
* ambos os processos podem realizar a mesma ação
* nenhum deles podem realizar a mesma ação 
Com essas características o protocolo não poderá existir, pois qualquer protocolo entre dois processos são equivalentes a uma série de troca de mensagens, e as premissas as impedem. E também as ações feitas por um processo dependem apenas da sequencia de mensagens que foi recebida, o que também não acontece de acordo com as premissas.


3. A prova apresentada convenceu você que o autor tem razão ? 
Perfeitamente. É como se duas pessoas fossem se comunicar mas as duas tem que falar exatamente ao mesmo tempo ou nao falar. Primeiro uma deve ouvir e depois a outra recebe a informação e depois fala.


4. Dê um exemplo de um sistema síncrono ?
Um exemplo de sistema síncrono é um sistema bancário, onde as transações são executadas e necessitam de uma resposta, ou uma ação imediata.


5. O que significa a frase "Postulating that a system is asynchronous is a non-assumption."  ? 
Significa que não podemos assumir que um sistema seja síncrono que mesmo que existam systemas síncronos, todos os sistemas são considerados assíncronos. 


6. Dê um exemplo de um sistema assíncrono (que não seja síncrono).
Um sistema de troca de mensagens, um chat. 


7.    Ao chegar à página 4, tente resolver o problema da eleição  SEM LER
os parágrafos seguintes !!!!  Ou seja, pare de ler na definição do problema.  
Discuta com seus colegas as possibilidades de solução.   Para que você controle sua curiosidade,
copio aqui o problema.    

Election Problem. A set of processes P1, P2, ..., Pn must select a leader. 
Each process Pi has a unique identifier uid(i).    Devise a protocol so that all of the processes 
learn the identity of the leader. Assume all processes start executing at the same time and
that all communicate using broadcasts that are reliable. 


Você deve propor duas soluções: 
       (i) uma supondo o modelo síncrono e 
       (ii) outra para o assíncrono.   
As soluções devem ser feitas em português, de forma organizada, 
dizendo o que ocorre entre os processos ao longo do tempo.  A única forma de comunicação é
troca de mensagens.

(i) O líder envia mensagens de broadcast para os seus liderados, já que a entrega é garantida e todos saberão que é o atual líder. Quando houver uma troca, uma nova mensagem de broadcast é transmitida e novamente todos os processos saberão quem é o novo leader.

(ii) O líder envia uma mensagem na rede, algum processo recebe a mensagem e transmite para seu vizinho, que faz a mesma coisa e assim transmitem a mensagem até todos receberem e todos saberem quem é o novo líder. Após essa parte um novo líder é eleito e repete o mesmo peocesso. 

8.  Você consegue exemplificar os diferentes tipos de defeitos listados na página 6 ?
Falhas bizantinas - invasão na rede e processos 

9.  Quais as relações entre sistemas distribuídos e tolerância a falhas, segundo o autor ?

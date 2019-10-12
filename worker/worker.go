package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/streadway/amqp"
)

var ipMaster = "http://172.18.0.3:2112"
var base *sql.DB

// Const referente as regras da mensageria
const (
	MsgTamInvalido = "Mensagem com tamanho de um dos bits invalidos"
	MsgConLetra    = "Mensagem contem um dos bits com letra(s)"
)

// const referente aos parametros de conexão com o PostgreSQL
const (
	host     = "172.18.0.2"
	port     = 5432
	user     = "postgres"
	password = "1234"
	dbname   = "microservicos"
	iprabbit = "172.18.0.4:5672"
)

func leituraFila(number int) {
	for {
		conn, channel, que := preparaRabbit("ExtressQueue")
		msgs, _ := channel.Consume(
			que.Name,
			"",
			true,
			false,
			false,
			false,
			nil,
		)

		for m := range msgs {
			mensagem := string(m.Body)

			retmsg, result := mapaMensageria(mensagem)
			log.Printf("Mensagem: %s worker: %d", retmsg, number)
			msgstatus := fmt.Sprint(retmsg + " Worker: " + strconv.Itoa(number))

			if result {
				InsertFilaSQL(mensagem, result)
			} else {
				InsertFilaSQL(msgstatus, result)
			}

		}
		conn.Close()
	}

}

// InsertFilaSQL função responsavel por inserir no PostgreSQL o resultado da mensageria
func InsertFilaSQL(mensagem string, result bool) {

	sqlStatement := `
	CREATE TABLE IF NOT EXISTS public.filarabbit
	(
		mensagem text COLLATE pg_catalog."default",
		resultado "char",
		datahora timestamp without time zone
	)
	WITH (
		OIDS = FALSE
	)
	`

	_, err := base.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}

	sqlStatement =
		`
	INSERT INTO FilaRabbit (mensagem, resultado, datahora)
	VALUES ($1, $2, $3)`
	horario := time.Now()
	_, err = base.Exec(sqlStatement, mensagem, result, horario.Format("2006-01-02 15:04:05"))
	if err != nil {
		panic(err)
	}

}

func mapaMensageria(msg string) (string, bool) {
	mensagem := ""
	f := func(c rune) bool {
		return c == ':' || c == ';'
	}

	// Separa a mensageria
	msgFields := strings.FieldsFunc(msg, f)
	for i := 0; i < len(msgFields); i++ {
		if len(msgFields[i]) != 4 {
			mensagem = MsgTamInvalido
			return mensagem, false
		}
		if strings.ContainsAny(msgFields[i], "abcdefghijklmnopqrstuvxz") {
			mensagem = MsgConLetra
			return mensagem, false
		}
	}

	mensagem = strings.Join(msgFields, " ")

	return mensagem, true
}

func preparaRabbit(queue string) (*amqp.Connection, *amqp.Channel, amqp.Queue) {
	conn, _ := amqp.Dial("amqp://guest:guest@172.18.0.4:5672")
	ch, _ := conn.Channel()

	q, _ := ch.QueueDeclare(
		queue, //name string,
		true,  // durable bool,
		false, // autodelete
		false, // exclusive
		false, // nowait
		nil)   // args

	ch.QueueBind(
		q.Name,       //name string,
		"",           //key string,
		"amq.fanout", //exchange string
		false,        //noWait bool,
		nil)          //args amqp.Table

	return conn, ch, q
}

//Worker de função para leitura/escrita de fila realtime
func main() {

	var ret *http.Response
	var err error
	//Enquanto não obter conexão não segue
	for {
		ret, err = http.Get(ipMaster + "/workers")
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	qntWorkers, err := ioutil.ReadAll(ret.Body)
	novqnt, _ := strconv.Atoi(string(qntWorkers))
	novqnt += 2112
	if err != nil {
		log.Fatalln(err)
	}

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	base = db // Depois de conectar e validar conexão, atribui para todos da rotina utilizar.

	go leituraFila(novqnt)
	port := ":" + strconv.Itoa(novqnt)
	println(port)

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(port, nil)

}

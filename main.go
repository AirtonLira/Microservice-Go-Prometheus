package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/streadway/amqp"
)

// Var para controle do total de workers ja instanciados.
var workers = 0

// Var responsavel para totalizar total de request do path sendmsg
var countrequestsend int

// Var compartilhada para instanciar o Postgresql
var base *sql.DB

// Var referente a metrica customizada do prometheus, armazenando total de chamadas do path sendmsg.
var requestSendmsg = promauto.NewCounter(prometheus.CounterOpts{
	Name: "main_total_request_sendmsg",
	Help: "Numer total de requisições do path sendmsg",
})

// Const referente as configurações do RabbitMQ e do PostgreSQL.
const (
	host     = "172.18.0.2"
	port     = 5432
	user     = "postgres"
	password = "1234"
	dbname   = "microservicos"
	iprabbit = "172.18.0.4:5672"
)

// PreparaRabbit para iniciar a conexão,canal e queue(fila) do rabbitmq
func preparaRabbit(queue string) (*amqp.Connection, *amqp.Channel, amqp.Queue) {
	conn, err := amqp.Dial("amqp://guest:guest@" + iprabbit)
	if err != nil {
		panic(err)
	}
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

//contWorkers - Função do handler para devolver quantidade de workers.
func contWorkers(w http.ResponseWriter, r *http.Request) {
	workers++
	strWorkers := strconv.Itoa(workers)
	w.Write([]byte(strWorkers))

	sqlStatement := `
	CREATE TABLE IF NOT EXISTS public.workersuse
	(
		numberworker integer,
		port integer,
		starttime timestamp without time zone
	)
	WITH (
		OIDS = FALSE
	)`
	_, err := base.Exec(sqlStatement)
	if err != nil {
		panic(err)
	}

	sqlStatement = `
	INSERT INTO workersuse (numberworker, port, starttime)
	VALUES ($1, $2, $3)`

	horario := time.Now()
	_, err = base.Exec(sqlStatement, workers, 2112+workers, horario.Format("2006-01-02 15:04:05"))
	if err != nil {
		panic(err)
	}

}
func metricscustom(w http.ResponseWriter, r *http.Request) {
	response := fmt.Sprintf("promhttp_metric_handler_requests_total{code=%q} ", "200")
	response = response + strconv.Itoa(countrequestsend)
	w.Write([]byte(response))
	return
}
func enviaMsg(w http.ResponseWriter, r *http.Request) {
	conn, channel, que := preparaRabbit("ExtressQueue")
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	texto := string(b)
	mensagem := fmt.Sprint(texto)
	msg := amqp.Publishing{
		Body: []byte(mensagem),
	}
	channel.Publish("", que.Name, false, false, msg)

	//Incrementa o counter da metrica custom criada
	requestSendmsg.Inc()

	conn.Close()

}

func main() {

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	time.Sleep(30 * time.Second)

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
	log.Println("Conectado ao postgres com sucesso!")

	http.HandleFunc("/workers", contWorkers)
	http.HandleFunc("/sendmsg", enviaMsg)
	http.Handle("/metrics", promhttp.Handler())
	log.Println("Serviço Master será iniciado na porta 2112")
	http.ListenAndServe(":2112", nil)

}

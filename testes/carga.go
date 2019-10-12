package main

import (
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

func main() {

	//Obtem o total de requisições a serem enviadas!
	quantparam := os.Args[2]
	log.Printf("total de requisições a enviar: %s", quantparam)
	qntrequest, _ := strconv.Atoi(quantparam)
	t1 := time.Now()
	for i := 0; i <= qntrequest; i++ {
		mensagem := fmt.Sprintf("%d:%d:%d:%d", rand.Intn(9999), rand.Intn(9999), rand.Intn(9999), rand.Intn(9999))
		_, err := http.Post("http://localhost:2112/sendmsg", "text/html", bytes.NewBufferString(mensagem))
		if err != nil {
			log.Fatalln(err)
		}
	}
	t2 := time.Now()
	diff := t2.Sub(t1)
	fmt.Printf("Diferença final: %s", diff)
}

package main 
//Web socket server dibuat menggunakan library Gorilla Web Socket
//frontend dengan native API milik JS dengan 4rd party library 
import (
	"fmt"
	"github.com/gorilla/websocket" //library websocket server 
	"github.com/novalagung/gubrak"
	"io/ioutil" 
	"log"
	"net/http" 
	"strings" //Dipakai untuk deteksi apakah string (parameter kedua) 
	//merupakan bagian dari string lain (parameter pertama). Nilai kembaliannya berupa bool.
)
// map
type M map[string]interface{} 
//konstanta dengan message dari socket server ke semua client (jenisnya)
const MESSAGE_NEW_USER = "New User" //broadcast user yang baru masuk
const MESSAGE_CHAT = "Chat" // mengirim pesan ke websocket
const MESSAGE_LEAVE = "Leave" //ketika ada yg keluar dari chat

var connections = make([]*WebSocketConnection, 0) //menampung semua client 

//struct SocketPayload 
//untuk menampung payload yang dikirim dari frontend
type SocketPayload struct{
	Message string
}
//backend mengirim pesan ke semua client
//membroadcast message ke semua client yang terhubung melalui konstanrta message
type SocketResponse struct{
	From string
	Type string
	Message string
}
//slice
//penyimpanan object untuk menghubungkan ke var connections
type WebSocketConnection struct{ 
	*websocket.Conn 
	Username string
}

func handleIO(currentConn *WebSocketConnection, connections []*WebSocketConnection) {
	defer func(){
		if r := recover();
		r != nil {
			log.Println("ERROR", fmt.Sprintf("%v", r))
		}
	}()

	//koneksi antara socket client dan server ketika koneksi pertama
	broadcastMessage(currentConn, MESSAGE_NEW_USER, "")

	for {
		payload := SocketPayload{}
		//perulangan tanpa henti dalam loop adalah blocking
		//hanya dieksekusi ketika ada payload atau pesan dari client trus di acc server
		err := currentConn.ReadJSON(&payload) //readjson agar koneksinya tidak terputus
		if err != nil{
			//eror jika terputus dengan socket server
			if strings.Contains(err.Error(), "websocket: close 1001") { //Dipakai untuk deteksi apakah string (parameter kedua) 
				//merupakan bagian dari string lain (parameter pertama). Nilai kembaliannya berupa bool.
				broadcastMessage(currentConn,MESSAGE_LEAVE, "") //untuk user leave room
				ejectConnection(currentConn)
				return
			}

			log.Println("EROR", err.Error())
			continue
		}
		//semua terhubung kecuali dengan jenis message chat
		broadcastMessage(currentConn, MESSAGE_CHAT, payload.Message)
	}
}
func ejectConnection(currentConn *WebSocketConnection){
	filtered, _ := gubrak.Reject(connections, func(each *WebSocketConnection) bool {
		return each == currentConn
	})
	connections = filtered.([]*WebSocketConnection)
}

//broadcastmessage dikirimi data kepada semua client
func broadcastMessage(currentConn *WebSocketConnection, kind, message string) {
	for _, eachConn := range connections{
		if eachConn == currentConn{
			continue
		}

		eachConn.WriteJSON(SocketResponse{  // mengirim data dari server ke client dengan melalui eachConn
			From: currentConn.Username,
			Type: kind,
			Message: message,
		})
	}
}
func main(){
	//menghubungkan dengan view index
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
		content, err := ioutil.ReadFile("index.html")
		if err != nil{
			http.Error(w,"File Undefined", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "%s", content)
	})

	//gateway komunikasi socket
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request){ 
		//socket code untuk konversi http ke web socket
		currentGorillaConn, err := websocket.Upgrade(w,r,w.Header(), 1024,1024) //besar read buffer dan besar write buffer
 		if err != nil{
			http.Error(w, "Could not open websocket", http.StatusBadRequest)
		}
		
		username := r.URL.Query().Get("username")
		//inisialisasi koneksi web socket dengan menyisipkan username sebagai query
		currentConn := WebSocketConnection{Conn: currentGorillaConn, Username: username} //sisi backend untuk ditepelkan ke objek 
		//koneksi socket
		//dimasukkan ke koneksi
		connections = append(connections, &currentConn) //slice

		//goroutine untuk memanage komunikasi antara client dan server
		go handleIO(&currentConn, connections) //goroutine
	})

	fmt.Println("server starting at :8080")
	http.ListenAndServe(":8080", nil)
}
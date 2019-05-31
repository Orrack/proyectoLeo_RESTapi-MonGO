package main

import (
	"encoding/json"
	"log"
	"net/http"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"github.com/gorilla/mux"
	//para obtener datos del config.toml
	"github.com/BurntSushi/toml"
	//librerias de tiempo
	"fmt"
	"strings"
	"time"
	//import math
)

//estructura para conexion
type Config struct {
	Server   string
	Database string
}

//leer y convertir el archivo configuracion
func (c *Config) Read() {
	if _, err := toml.DecodeFile("config.toml", &c); err != nil {
		log.Fatal(err)
	}
}

//estructura de apoyo para el server (para evitar confusion)
type botonDAO struct {
	Server   string
	Database string
}

var db *mgo.Database

//nombre de la collection *se puede ajustar*
const (
	COLLECTION = "sensores"
	TOKENS = "permisos"
)

// Establecer coneccion a BD mongo
func (m *botonDAO) Connect() {
	session, err := mgo.Dial(m.Server)
	if err != nil {
		log.Fatal(err)
	}
	db = session.DB(m.Database)
}

//------------------------------------------------------------------------------
func (m *botonDAO) FindToken(tok string) ([]NODEMCU, error) {
	var node []NODEMCU
	err := db.C(TOKENS).Find(bson.M{"token":tok,"status":"OK"}).All(&node)
	return node, err
}

// buscar a todos los registros
func (m *botonDAO) FindAll() ([]BOTON, error) {
	var btn []BOTON
	err := db.C(COLLECTION).Find(bson.M{}).All(&btn)
	return btn, err
}

// Buscar por id del registro
func (m *botonDAO) FindById(id string) (BOTON, error) {
	var btn BOTON
	err := db.C(COLLECTION).FindId(bson.ObjectIdHex(id)).One(&btn)
	return btn, err
}

// insertar registro a la BD mongo
func (m *botonDAO) Insert(btn BOTON) error {
	err := db.C(COLLECTION).Insert(&btn)
	return err
}

// eliminar registro de BD
func (m *botonDAO) Delete(btn BOTON) error {
	err := db.C(COLLECTION).Remove(&btn)
	return err
}

// Actualizar en base al id
func (m *botonDAO) Update(btn BOTON) error {
	err := db.C(COLLECTION).UpdateId(btn.ID, &btn)
	return err
}

//------------------------------------------------------------------------------

//estructura de json para la BD
type BOTON struct {
	ID          	bson.ObjectId 	`bson:"_id" json:"id"`
	H   		float64        	`bson:"humedad" json:"humedad"`
	T		float64        	`bson:"temperatura" json:"temperatura"`
	V		float64 	`bson:"velAire" json:"velAire"`
	Fecha		string		`bson:"fecha" json:"fecha"`
	Hora		string		`bson:"hora" json:"hora"`
}
type NODEMCU struct {
	ID          	bson.ObjectId `bson:"_id" json:"id"`
	D   		string        `bson:"dispositivo" json:"dispositivo"`
	T		string        `bson:"token" json:"token"`
	S		string        `bson:"status" json:"status"`
}


//reciclan variables para proximos metodos
var config = Config{}
var dao = botonDAO{}

// Peticion GET para obtener a todos (hace referencia al metodo FindAll)
func All_Reg(w http.ResponseWriter, r *http.Request) {
	btn, err := dao.FindAll()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJson(w, http.StatusOK, btn)
}

// Peticion Get para buscar registro
func Find_Reg(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	btn, err := dao.FindById(params["id"])
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "No existe ID (numero identificador)")
		return
	}
	respondWithJson(w, http.StatusOK, btn)
}

// Peticion Post para insertar valor
func Create_Reg(w http.ResponseWriter, r *http.Request) {
	x := r.Header.Get("Authorization")
	auth := strings.Replace(x,"Basic ","",-1)
	defer r.Body.Close()
	var btn BOTON
	node, err := dao.FindToken(auth)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "No existe Token")
		return
	}
		if node != nil{
			if err := json.NewDecoder(r.Body).Decode(&btn); err != nil {
				respondWithError(w, http.StatusBadRequest, "Invalid request payload")
				return
			}
			btn.ID = bson.NewObjectId()
			t := time.Now()
			fec :=""+fmt.Sprintf("%d-%02d-%02d",t.Year(), t.Month(), t.Day())
			hor := ""+fmt.Sprintf("%02d:%02d:%02d",t.Hour(), t.Minute(), t.Second())
			btn.Fecha=fec
			btn.Hora=hor
			if err := dao.Insert(btn); err != nil {
				respondWithError(w, http.StatusInternalServerError, err.Error())
				return
			}
			respondWithJson(w, http.StatusCreated, btn)
	 		}
}

// peticion put para actualizar registro
func Update_Reg(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var btn BOTON
	if err := json.NewDecoder(r.Body).Decode(&btn); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := dao.Update(btn); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJson(w, http.StatusOK, map[string]string{"result": "success"})
}

// Eliminar registro de bd
func Delete_Reg(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var btn BOTON
	if err := json.NewDecoder(r.Body).Decode(&btn); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if err := dao.Delete(btn); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJson(w, http.StatusOK, map[string]string{"result": "success"})
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	respondWithJson(w, code, map[string]string{"error": msg})
}

func respondWithJson(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

// configuracion de servidor y base de datos , Config y botonDAO toman valores de aqui
func init() {
	config.Read()
	dao.Server = config.Server
	dao.Database = config.Database
	dao.Connect()
}

// definicion de rutas para api
func main() {
	r := mux.NewRouter()

	r.HandleFunc("/sensor", All_Reg).Methods("GET")
	r.HandleFunc("/sensor", Create_Reg).Methods("POST")
	//r.HandleFunc("/boton", Update_Reg).Methods("PUT")
	//r.HandleFunc("/boton", Delete_Reg).Methods("DELETE")
	r.HandleFunc("/sensor/{id}", Find_Reg).Methods("GET")
	if err := http.ListenAndServe(":3312", r); err != nil {
		log.Fatal(err)
	}
}

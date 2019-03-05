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
	COLLECTION = "regBoton"
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
	ID          bson.ObjectId `bson:"_id" json:"id"`
	Name        string        `bson:"name" json:"name"`
	Carac 			string        `bson:"carac" json:"carac"`
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
	defer r.Body.Close()
	var btn BOTON
	if err := json.NewDecoder(r.Body).Decode(&btn); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	btn.ID = bson.NewObjectId()
	if err := dao.Insert(btn); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondWithJson(w, http.StatusCreated, btn)
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
	r.HandleFunc("/boton", All_Reg).Methods("GET")
	r.HandleFunc("/boton", Create_Reg).Methods("POST")
	//r.HandleFunc("/boton", Update_Reg).Methods("PUT")
	//r.HandleFunc("/boton", Delete_Reg).Methods("DELETE")
	r.HandleFunc("/boton/{id}", Find_Reg).Methods("GET")
	if err := http.ListenAndServe(":3000", r); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"encoding/json"
    "log"
	"html/template"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	//"github.com/gorilla/websocket"
)

type Project struct {
    Name  string `json:"name"`
    Url   string `json:"url"`
    Img   string `json:"img"`
}

func home(c *gin.Context) {
    tmpl, err := template.New("page").ParseFiles("html/base.html", "html/home.html")
    if err != nil {
        c.Status(http.StatusInternalServerError)
        log.Println(err)
        return
    }
    var projects []Project
    jsonFile, err := os.Open("static/projects.json")
    if err != nil {
        c.Status(http.StatusInternalServerError)
        log.Println(err)
        return
    }
    decode := json.NewDecoder(jsonFile)
    decode.Decode(&projects)
    tmpl.Execute(c.Writer, projects)
}

func gis(c *gin.Context) {
    tmpl, err := template.New("page").ParseFiles("html/base.html", "html/gis.html")
    if err != nil {
        c.Status(http.StatusInternalServerError)
        log.Println(err)
        return
    }
    tmpl.Execute(c.Writer, nil)
}

func main() {
    defer browser.Close()

    gin.SetMode(gin.ReleaseMode)
    r := gin.Default()

    r.Static("/static", "./static")
    r.StaticFile("/favicon.ico", "./static/favicon.ico")
    r.StaticFile("/robots.txt", "./static/robots.txt")
    r.StaticFile("/sitemap.txt", "./static/sitemap.txt")
    r.StaticFile("/wasm", "./html/host.html")

    r.GET("/", home)
    r.GET("/lexos", lexos)
    r.GET("/ws", ws)
    r.GET("/gis", gis)
    
    log.Println("Starting server on port 80")
    err := r.Run(":80")
    if err != nil {
        panic(err)
    }
}
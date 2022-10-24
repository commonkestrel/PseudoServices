package main

import (
    "errors"
    "fmt"
    "strings"
    "net/http"
    "html/template"

    isbnpkg "github.com/moraes/isbn"
    "github.com/playwright-community/playwright-go"
    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
)

type Result struct {
    Lexile int     `json:"lexile"`
    Atos   float64 `json:"atos"`
    Ar     float64 `json:"ar"`
}

const (
    lexile_url = "https://hub.lexile.com/find-a-book/book-details/"
    lexile_selector = "#content > div > div > div > div.details > div.metadata > div.sc-kexyCK.cawTwh > div.header-info > div > span"
    
    atos_url = "https://www.arbookfind.com/UserType.aspx?RedirectURL=%2fadvanced.aspx"
    rad = "#radLibrarian"
    submit = "#btnSubmitUserType"
    isbn_box = "#ctl00_ContentPlaceHolder1_txtISBN"
    search = "#ctl00_ContentPlaceHolder1_btnDoIt"
    search_fail = "#ctl00_ContentPlaceHolder1_lblSearchResultFailedLabel"
    title = "#book-title"
    atos_level = "#ctl00_ContentPlaceHolder1_ucBookDetail_lblBookLevel"
    ar_points = "#ctl00_ContentPlaceHolder1_ucBookDetail_lblPoints"
)

var (
    pw *playwright.Playwright
    browser playwright.Browser

    errInvalidIsbn = errors.New("invalid isbn")
    upgrader = websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
            return true
        },
    }
)

func init() {
    var err error
    pw, err = playwright.Run()
    if err != nil { panic(err) }

    browser, err = pw.Chromium.Launch()
    if err != nil { panic(err) }
}

func get(isbn string) (Result, error) {
    isbn = strings.ReplaceAll(isbn, "-", "")
    valid := isbnpkg.Validate(isbn)
    if !valid {
        return Result{}, errInvalidIsbn
    }

    page, err := browser.NewPage()
    if err != nil {
        return Result{}, err
    }
    defer page.Close()

    lex := lexile(page, isbn)
    at, ar := atos(page, isbn)

    return Result{lex, at, ar}, nil
}

func lexile(page playwright.Page, isbn string) int {
    page.Goto(fmt.Sprint(lexile_url, isbn))
    if page.URL() == "https://hub.lexile.com/find-a-book/book-results" {
        return -1
    }

    str, err := page.TextContent(lexile_selector)
    if err != nil {
        return -1
    }
    var lex int
    if _, err := fmt.Sscan(str, &lex); err != nil {
        return -1
    }
    return lex
}

func atos(page playwright.Page, isbn string) (float64, float64) {
    page.Goto(atos_url)
    page.Click(rad) //Select Librarian and submit
    page.Click(submit)

    page.WaitForSelector(isbn_box)
    page.Type(isbn_box, isbn)
    page.Click(search)
    
    page.WaitForLoadState("domcontentloaded")
    fail, _ := page.Locator(search_fail)
    count, _ := fail.Count()
    if count > 0 {
        return -1, -1
    }

    page.WaitForSelector(title)
    page.Click(title) //Click on first book

    var atos float64
    var ar float64
    AtosStr, err := page.TextContent(atos_level) //Get level from selector
    if err != nil {
        AtosStr = "-1"
    }
    ArStr, err := page.TextContent(ar_points)
    if err != nil {
        ArStr = "-1"
    }
    
    fmt.Sscan(ArStr, &ar)
    fmt.Sscan(AtosStr, &atos)
    return atos, ar
}

func lexos(c *gin.Context) {
    tmpl, err := template.New("page").ParseFiles("html/base.html", "html/lexos.html")
    if err != nil {
        c.Status(http.StatusInternalServerError)
        return
    }
    tmpl.Execute(c.Writer, nil)
}

func ws(c *gin.Context) {
    isbn := c.Query("isbn")
    if isbn == "" {
        c.Status(http.StatusBadRequest)
        return
    }

    ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        c.Status(http.StatusInternalServerError)
        return
    }
    defer ws.Close()

    res, err := get(isbn)
    if err != nil {
        ws.WriteMessage(websocket.TextMessage, []byte("error:" + err.Error()))
        return
    }
    ws.WriteJSON(res)
}

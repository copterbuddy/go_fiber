package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	jwtware "github.com/gofiber/jwt/v3"
	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

var db *sqlx.DB

const jwtSecret = "mySecret"

func main() {

	var err error
	db, err = sqlx.Open("mysql", "root:P@ssw0rd@tcp(127.0.0.1:3306)/banking")
	if err != nil {
		panic(err)
	}

	app := fiber.New()

	app.Use("/hello", jwtware.New(jwtware.Config{
		SigningMethod: "HS256",
		SigningKey:    []byte(jwtSecret),
		SuccessHandler: func(c *fiber.Ctx) error {
			return c.Next()
		},
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return fiber.ErrUnauthorized
		},
	}))

	app.Post("/signup", Signup)
	app.Post("/login", Login)
	app.Get("/hello", Hello)

	app.Listen(":8000")
}

func Signup(c *fiber.Ctx) error {

	request := SignupRequest{}
	err := c.BodyParser(&request)
	if err != nil {
		return err
	}

	if request.Username == "" || request.Password == "" {
		return fiber.ErrUnprocessableEntity
	}

	password, err := bcrypt.GenerateFromPassword([]byte(request.Password), 10)
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}

	query := "insert user (username,password) values(?,?)"
	result, err := db.Exec(query, request.Username, string(password))
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fiber.NewError(fiber.StatusUnprocessableEntity, err.Error())
	}

	user := User{
		Id:       int(id),
		Username: request.Username,
		Password: string(password),
	}
	return c.Status(fiber.StatusCreated).JSON(user)
}

func Login(c *fiber.Ctx) error {

	request := LoginRequest{}
	err := c.BodyParser(&request)
	if err != nil {
		return err
	}

	if request.Username == "" || request.Password == "" {
		return fiber.ErrUnprocessableEntity
	}

	user := User{}
	query := "select id,username,password from user where username=?"
	err = db.Get(&user, query, request.Username)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "incorrect username or password")
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(request.Password))
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "incorrect username or password")
	}

	cliams := jwt.StandardClaims{
		Issuer:    strconv.Itoa(user.Id),
		ExpiresAt: time.Now().Add(time.Hour * 1).Unix(),
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, cliams)
	token, err := jwtToken.SignedString([]byte(jwtSecret))
	if err != nil {
		return fiber.NewError(fiber.StatusUnauthorized)
	}

	return c.JSON(fiber.Map{
		"jwtToken": token,
	})
}

func Hello(c *fiber.Ctx) error {
	return c.SendString("Hello World")
}

type User struct {
	Id       int    `db:"id" json:"id"`
	Username string `db:"username" json:"username"`
	Password string `db:"password" json:"password"`
}

type SignupRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func Fiber() {
	app := fiber.New(fiber.Config{
		Prefork:       true,
		CaseSensitive: false,
		StrictRouting: false,
	})

	//Middleware
	app.Use("/hello", func(c *fiber.Ctx) error {
		c.Locals("name", "bond")
		fmt.Println("before")
		err := c.Next()
		fmt.Println("after")
		return err
	})

	app.Use(requestid.New())

	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "*",
		AllowHeaders: "*",
	}))

	// app.Use(logger.New(logger.Config{
	// 	TimeZone: "Asia/Bangkok",
	// }))

	//GET
	app.Get("/hello", func(c *fiber.Ctx) error {
		name := c.Locals("name")
		return c.SendString(fmt.Sprintf("GET: Hello World %v", name))
	})

	//POST
	app.Post("/hello", func(c *fiber.Ctx) error {
		return c.SendString("POST: Hello World")
	})

	//Parameters
	app.Get("/hello/:name/:surname?", func(c *fiber.Ctx) error {
		name := c.Params("name")
		surname := c.Params("surname")
		return c.SendString("name: " + name + ", surname: " + surname)
	})

	//ParamInt
	app.Get("/hello2/:id", func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return fiber.ErrBadRequest
		}

		return c.SendString(fmt.Sprintf("ID = %v", id))
	})

	//Query
	app.Get("/query", func(c *fiber.Ctx) error {
		name := c.Query("name")
		surname := c.Query("surname")
		return c.SendString("name: " + name + " surname: " + surname)
	})

	//Query
	app.Get("/query2", func(c *fiber.Ctx) error {
		person := Person{}
		c.QueryParser(&person)
		return c.JSON(person)
	})

	//Wildcard
	app.Get("/wildcards/*", func(c *fiber.Ctx) error {
		wildcard := c.Params("*")
		return c.SendString(wildcard)
	})

	//Static file
	app.Static("/", "./wwwroot", fiber.Static{
		Index:         "index.html",
		CacheDuration: time.Second * 10,
	})

	//NewError
	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "content not found")
	})

	//Group
	v1 := app.Group("/v1")
	v1.Get("/hello", func(c *fiber.Ctx) error {
		c.Set("Version", "v1")
		return c.Next()
	})

	v2 := app.Group("/v2")
	v2.Get("/hello", func(c *fiber.Ctx) error {
		c.Set("Version", "v2")
		c.Get("Version")
		return c.Next()
	})

	//Mount
	userApp := fiber.New()
	userApp.Get("/login", func(c *fiber.Ctx) error {
		return c.SendString("Login")
	})

	app.Mount("/user", userApp)

	//Server
	app.Server().MaxConnsPerIP = 1
	app.Get("/server", func(c *fiber.Ctx) error {
		time.Sleep(time.Second * 3)
		return c.SendString("server")
	})

	app.Get("/env", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"BaseURL":     c.BaseURL(),
			"Hostname":    c.Hostname(),
			"IP":          c.IP(),
			"IPs":         c.IPs(),
			"OriginalURL": c.OriginalURL(),
			"Path":        c.Path(),
			"Protocal":    c.Protocol(),
			"Subdomains":  c.Subdomains(),
		})
	})

	//Body
	app.Post("/body", func(c *fiber.Ctx) error {
		fmt.Printf("IsJson: %v", c.Is("json"))
		fmt.Println()
		// fmt.Println(string(c.Body()))

		person := Person{}
		err := c.BodyParser(&person)
		if err != nil {
			return err
		}
		fmt.Println(person)
		return nil
	})

	//Body2
	app.Post("/body2", func(c *fiber.Ctx) error {
		fmt.Printf("IsJson: %v", c.Is("json"))
		fmt.Println()
		// fmt.Println(string(c.Body()))

		person := map[string]interface{}{}
		err := c.BodyParser(&person)
		if err != nil {
			return err
		}

		fmt.Println(person)
		return nil
	})

	app.Listen(":8000")
}

type Person struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

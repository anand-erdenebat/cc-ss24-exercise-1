package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Defines a "model" that we can use to communicate with the
// frontend or the database
// More on these "tags" like `bson:"_id,omitempty"`: https://go.dev/wiki/Well-known-struct-tags
type BookStore struct {
	MongoID     primitive.ObjectID `bson:"_id,omitempty"`
	ID          string
	BookName    string
	BookAuthor  string
	BookEdition string
	BookPages   string
	BookYear    string
}

// Wraps the "Template" struct to associate a necessary method
// to determine the rendering procedure
type Template struct {
	tmpl *template.Template
}

// Preload the available templates for the view folder.
// This builds a local "database" of all available "blocks"
// to render upon request, i.e., replace the respective
// variable or expression.
// For more on templating, visit https://jinja.palletsprojects.com/en/3.0.x/templates/
// to get to know more about templating
// You can also read Golang's documentation on their templating
// https://pkg.go.dev/text/template
func loadTemplates() *Template {
	return &Template{
		tmpl: template.Must(template.ParseGlob("views/*.html")),
	}
}

// Method definition of the required "Render" to be passed for the Rendering
// engine.
// Contraire to method declaration, such syntax defines methods for a given
// struct. "Interfaces" and "structs" can have methods associated with it.
// The difference lies that interfaces declare methods whether struct only
// implement them, i.e., only define them. Such differentiation is important
// for a compiler to ensure types provide implementations of such methods.
func (t *Template) Render(w io.Writer, name string, data interface{}, ctx echo.Context) error {
	return t.tmpl.ExecuteTemplate(w, name, data)
}

// Here we make sure the connection to the database is correct and initial
// configurations exists. Otherwise, we create the proper database and collection
// we will store the data.
// To ensure correct management of the collection, we create a return a
// reference to the collection to always be used. Make sure if you create other
// files, that you pass the proper value to ensure communication with the
// database
// More on what bson means: https://www.mongodb.com/docs/drivers/go/current/fundamentals/bson/
func prepareDatabase(client *mongo.Client, dbName string, collecName string) (*mongo.Collection, error) {
	db := client.Database(dbName)

	names, err := db.ListCollectionNames(context.TODO(), bson.D{{}})
	if err != nil {
		return nil, err
	}
	if !slices.Contains(names, collecName) {
		cmd := bson.D{{"create", collecName}}
		var result bson.M
		if err = db.RunCommand(context.TODO(), cmd).Decode(&result); err != nil {
			log.Fatal(err)
			return nil, err
		}
	}

	coll := db.Collection(collecName)
	return coll, nil
}

// Here we prepare some fictional data and we insert it into the database
// the first time we connect to it. Otherwise, we check if it already exists.
func prepareData(client *mongo.Client, coll *mongo.Collection) {
	startData := []BookStore{
		{
			ID:          "example1",
			BookName:    "The Vortex",
			BookAuthor:  "José Eustasio Rivera",
			BookEdition: "958-30-0804-4",
			BookPages:   "292",
			BookYear:    "1924",
		},
		{
			ID:          "example2",
			BookName:    "Frankenstein",
			BookAuthor:  "Mary Shelley",
			BookEdition: "978-3-649-64609-9",
			BookPages:   "280",
			BookYear:    "1818",
		},
		{
			ID:          "example3",
			BookName:    "The Black Cat",
			BookAuthor:  "Edgar Allan Poe",
			BookEdition: "978-3-99168-238-7",
			BookPages:   "280",
			BookYear:    "1843",
		},
	}

	// This syntax helps us iterate over arrays. It behaves similar to Python
	// However, range always returns a tuple: (idx, elem). You can ignore the idx
	// by using _.
	// In the topic of function returns: sadly, there is no standard on return types from function. Most functions
	// return a tuple with (res, err), but this is not granted. Some functions
	// might return a ret value that includes res and the err, others might have
	// an out parameter.
	for _, book := range startData {
		cursor, err := coll.Find(context.TODO(), book)
		var results []BookStore
		if err = cursor.All(context.TODO(), &results); err != nil {
			panic(err)
		}
		if len(results) > 1 {
			log.Fatal("more records were found")
		} else if len(results) == 0 {
			result, err := coll.InsertOne(context.TODO(), book)
			if err != nil {
				panic(err)
			} else {
				fmt.Printf("%+v\n", result)
			}

		} else {
			for _, res := range results {
				cursor.Decode(&res)
				fmt.Printf("%+v\n", res)
			}
		}
	}
}

// Generic method to perform "SELECT * FROM BOOKS" (if this was SQL, which
// it is not :D ), and then we convert it into an array of map. In Golang, you
// define a map by writing map[<key type>]<value type>{<key>:<value>}.
// interface{} is a special type in Golang, basically a wildcard...
func findAllBooks(coll *mongo.Collection) []map[string]interface{} {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	var ret []map[string]interface{}
	for _, res := range results {
		ret = append(ret, map[string]interface{}{
			"ID":          res.MongoID.Hex(),
			"BookName":    res.BookName,
			"BookAuthor":  res.BookAuthor,
			"BookEdition": res.BookEdition,
			"BookPages":   res.BookPages,
		})
	}

	return ret
}

func findAllAuthors(coll *mongo.Collection) []map[string]interface{} {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	var ret []map[string]interface{}
	for _, res := range results {
		ret = append(ret, map[string]interface{}{
			"ID":         res.MongoID.Hex(),
			"BookAuthor": res.BookAuthor,
		})
	}

	return ret
}

func findAllYears(coll *mongo.Collection) []map[string]interface{} {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	var ret []map[string]interface{}
	for _, res := range results {
		ret = append(ret, map[string]interface{}{
			"ID":       res.MongoID.Hex(),
			"BookYear": res.BookYear,
		})
	}

	return ret
}

func findAllBooksAPI(coll *mongo.Collection) []map[string]interface{} {
	cursor, err := coll.Find(context.TODO(), bson.D{{}})
	var results []BookStore
	if err = cursor.All(context.TODO(), &results); err != nil {
		panic(err)
	}

	var ret []map[string]interface{}
	for _, res := range results {
		ret = append(ret, map[string]interface{}{
			"id":      res.ID,
			"title":   res.BookName,
			"author":  res.BookAuthor,
			"edition": res.BookEdition,
			"pages":   res.BookPages,
			"year":    res.BookYear,
		})
	}

	return ret
}

func createBook(c echo.Context, coll *mongo.Collection) error {
	// Define struct to bind the incoming request
	var input struct {
		ID          string `json:"id"`
		BookName    string `json:"title"`
		BookAuthor  string `json:"author"`
		BookEdition string `json:"edition"`
		BookPages   string `json:"pages"`
		BookYear    string `json:"year"`
	}

	// Bind the request body
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Invalid input",
		})
	}

	// Validate required fields
	if input.ID == "" || input.BookName == "" || input.BookAuthor == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"error": "Missing required fields",
		})
	}

	// Create a BookStore instance
	book := BookStore{
		MongoID:     primitive.NewObjectID(),
		ID:          input.ID,
		BookName:    input.BookName,
		BookAuthor:  input.BookAuthor,
		BookEdition: input.BookEdition,
		BookPages:   input.BookPages,
		BookYear:    input.BookYear,
	}

	// Insert into MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := coll.InsertOne(ctx, book)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"error": "Failed to insert book",
		})
	}

	// Return minimal JSON (like your GET function)
	response := map[string]interface{}{
		"id":      book.ID,
		"title":   book.BookName,
		"author":  book.BookAuthor,
		"edition": book.BookEdition,
		"pages":   book.BookPages,
		"year":    book.BookYear,
	}

	return c.JSON(http.StatusCreated, response)
}

func updateBook(c echo.Context, coll *mongo.Collection) error {
	// Get the book ID from the URL param
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Missing book ID in URL"})
	}

	var input struct {
		BookName    *string `json:"title,omitempty"`
		BookAuthor  *string `json:"author,omitempty"`
		BookPages   *string `json:"pages,omitempty"`
		BookEdition *string `json:"edition,omitempty"`
		BookYear    *string `json:"year,omitempty"`
	}

	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Invalid input"})
	}

	// Build update document with provided fields only
	updateData := bson.M{}
	if input.BookName != nil {
		updateData["BookName"] = *input.BookName
	}
	if input.BookAuthor != nil {
		updateData["BookAuthor"] = *input.BookAuthor
	}
	if input.BookPages != nil {
		updateData["BookPages"] = *input.BookPages
	}
	if input.BookEdition != nil {
		updateData["BookEdition"] = *input.BookEdition
	}
	if input.BookYear != nil {
		updateData["BookYear"] = *input.BookYear
	}

	if len(updateData) == 0 {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "No fields to update"})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"ID": id}

	var book BookStore
	err := coll.FindOne(ctx, filter).Decode(&book)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return c.JSON(http.StatusNotFound, echo.Map{"error": "Book not found"})
		}
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to find book"})
	}

	update := bson.M{"$set": updateData}
	_, err = coll.UpdateOne(ctx, filter, update)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to update book"})
	}

	var updatedBook BookStore
	err = coll.FindOne(ctx, filter).Decode(&updatedBook)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to fetch updated book"})
	}

	response := map[string]interface{}{
		"id":      updatedBook.ID,
		"title":   updatedBook.BookName,
		"author":  updatedBook.BookAuthor,
		"edition": updatedBook.BookEdition,
		"pages":   updatedBook.BookPages,
		"year":    updatedBook.BookYear,
	}

	return c.JSON(http.StatusOK, response)
}

func deleteBook(c echo.Context, coll *mongo.Collection) error {
	ctx := context.TODO()

	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"error": "Missing book ID"})
	}

	filter := bson.M{"ID": id}

	result, err := coll.DeleteOne(ctx, filter)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"error": "Failed to delete book"})
	}

	if result.DeletedCount == 0 {
		return c.JSON(http.StatusNotFound, echo.Map{"error": "Book not found"})
	}

	return c.JSON(http.StatusOK, echo.Map{"message": "Book deleted successfully"})
}

func main() {
	// Connect to the database. Such defer keywords are used once the local
	// context returns; for this case, the local context is the main function
	// By user defer function, we make sure we don't leave connections
	// dangling despite the program crashing. Isn't this nice? :D
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// TODO: make sure to pass the proper username, password, and port
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))

	// This is another way to specify the call of a function. You can define inline
	// functions (or anonymous functions, similar to the behavior in Python)
	defer func() {
		if err = client.Disconnect(ctx); err != nil {
			panic(err)
		}
	}()

	// You can use such name for the database and collection, or come up with
	// one by yourself!
	coll, err := prepareDatabase(client, "exercise-1", "information")

	prepareData(client, coll)

	// Here we prepare the server
	e := echo.New()

	// Define our custom renderer
	e.Renderer = loadTemplates()

	// Log the requests. Please have a look at echo's documentation on more
	// middleware
	e.Use(middleware.Logger())

	e.Static("/css", "css")

	// Endpoint definition. Here, we divided into two groups: top-level routes
	// starting with /, which usually serve webpages. For our RESTful endpoints,
	// we prefix the route with /api to indicate more information or resources
	// are available under such route.
	e.GET("/", func(c echo.Context) error {
		return c.Render(200, "index", nil)
	})

	e.GET("/books", func(c echo.Context) error {
		books := findAllBooks(coll)
		return c.Render(200, "book-table", books)
	})

	e.GET("/authors", func(c echo.Context) error {
		authors := findAllAuthors(coll)
		return c.Render(200, "authors", authors)
	})

	e.GET("/years", func(c echo.Context) error {
		years := findAllYears(coll)
		return c.Render(200, "years", years)
	})

	e.GET("/search", func(c echo.Context) error {
		return c.Render(200, "search-bar", nil)
	})

	e.GET("/create", func(c echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	// You will have to expand on the allowed methods for the path
	// `/api/route`, following the common standard.
	// A very good documentation is found here:
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Methods
	// It specifies the expected returned codes for each type of request
	// method.
	e.GET("/api/books", func(c echo.Context) error {
		books := findAllBooksAPI(coll)
		return c.JSON(http.StatusOK, books)
	})

	e.POST("/api/books", func(c echo.Context) error {
		books := createBook(c, coll)
		return c.JSON(http.StatusOK, books)
	})

	e.PUT("/api/books/:id", func(c echo.Context) error {
		books := updateBook(c, coll)
		return c.JSON(http.StatusOK, books)
	})

	e.DELETE("/api/books/:id", func(c echo.Context) error {
		return deleteBook(c, coll)
	})

	// We start the server and bind it to port 3030. For future references, this
	// is the application's port and not the external one. For this first exercise,
	// they could be the same if you use a Cloud Provider. If you use ngrok or similar,
	// they might differ.
	// In the submission website for this exercise, you will have to provide the internet-reachable
	// endpoint: http://<host>:<external-port>
	e.Logger.Fatal(e.Start(":3030"))
}

package main

type Task struct {
	ID string `json:"id"`
	Title string `json:"title"`
	CreatedAt `json:"created_at"`
	UpdatedAt `json:"updated_at"`

}

func main() {
	fmt.Print("lolo")
}
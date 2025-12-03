// MVC Example - Minimal MVC pattern using IoC container
//
// This example demonstrates a simple MVC architecture:
// - Model: User data structure
// - View: HTML template rendering
// - Controller: Request handling with injected services
package main

import (
	"fmt"

	ioc "github.com/MunMunMiao/go-ioc"
)

// ============================================================================
// Model Layer
// ============================================================================

type User struct {
	ID    int
	Name  string
	Email string
}

// ============================================================================
// Repository (Data Access)
// ============================================================================

type UserRepository struct {
	users []User
}

func (r *UserRepository) FindByID(id int) *User {
	for _, u := range r.users {
		if u.ID == id {
			return &u
		}
	}
	return nil
}

func (r *UserRepository) FindAll() []User {
	return r.users
}

var UserRepositoryRef = ioc.Provide(func(ctx *ioc.Context) *UserRepository {
	return &UserRepository{
		users: []User{
			{ID: 1, Name: "Alice", Email: "alice@example.com"},
			{ID: 2, Name: "Bob", Email: "bob@example.com"},
			{ID: 3, Name: "Charlie", Email: "charlie@example.com"},
		},
	}
})

// ============================================================================
// View Layer
// ============================================================================

type View struct{}

func (v *View) RenderUser(user *User) string {
	if user == nil {
		return "<div class='error'>User not found</div>"
	}
	return fmt.Sprintf("<div class=\"user\">\n  <h2>%s</h2>\n  <p>Email: %s</p>\n</div>", user.Name, user.Email)
}

func (v *View) RenderUserList(users []User) string {
	html := "<ul class='user-list'>\n"
	for _, u := range users {
		html += fmt.Sprintf("  <li>%s (%s)</li>\n", u.Name, u.Email)
	}
	html += "</ul>"
	return html
}

var ViewRef = ioc.Provide(func(ctx *ioc.Context) *View {
	return &View{}
})

// ============================================================================
// Controller Layer
// ============================================================================

type UserController struct {
	repo *UserRepository
	view *View
}

func (c *UserController) Show(id int) string {
	user := c.repo.FindByID(id)
	return c.view.RenderUser(user)
}

func (c *UserController) Index() string {
	users := c.repo.FindAll()
	return c.view.RenderUserList(users)
}

var UserControllerRef = ioc.Provide(func(ctx *ioc.Context) *UserController {
	return &UserController{
		repo: ioc.Inject(ctx, UserRepositoryRef),
		view: ioc.Inject(ctx, ViewRef),
	}
})

// ============================================================================
// Application Entry Point
// ============================================================================

func main() {
	ioc.RunInInjectionContext(func(ctx *ioc.Context) any {
		controller := ioc.Inject(ctx, UserControllerRef)

		fmt.Println("=== User List ===")
		fmt.Println(controller.Index())

		fmt.Println("\n=== Single User ===")
		fmt.Println(controller.Show(1))

		fmt.Println("\n=== User Not Found ===")
		fmt.Println(controller.Show(999))

		return nil
	})
}

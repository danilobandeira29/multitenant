package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

type Role int

const (
	Admin Role = iota
	User
)

func (r Role) toString() string {
	return [...]string{"Admin", "User"}[r]
}

type Permission int

const (
	CreateUser Permission = iota
	UpdateUser
	DeleteUser
	GivePermission
	WatchContent
)

func (p Permission) toString() string {
	return [...]string{
		"CreateUser",
		"UpdateUser",
		"DeleteUser",
		"GivePermission",
		"WatchContent",
	}[p]
}

type UserEntity struct {
	ID            string
	Name          string
	DeactivatedAt *time.Time
	Roles         map[string]map[Role][]Permission
}

var users = []UserEntity{
	{
		ID:            "1",
		Name:          "Danilo Bandeira",
		DeactivatedAt: nil,
		Roles: map[string]map[Role][]Permission{
			"Product1": {
				Admin: {
					CreateUser,
					DeleteUser,
					UpdateUser,
					GivePermission,
				},
			},
			"Product2": {
				User: {
					WatchContent,
				},
			},
		},
	},
	{
		ID:            "2",
		Name:          "Ana Banana",
		DeactivatedAt: nil,
		Roles: map[string]map[Role][]Permission{
			"Product1": {
				User: {
					WatchContent,
				},
			},
			"Product2": {
				User: {
					WatchContent,
				},
			},
		},
	},
}

func hasPermission(entity UserEntity, feature string, product string) bool {
	roles, ok := entity.Roles[product]
	if !ok {
		return false
	}
	for _, permissions := range roles {
		for _, perm := range permissions {
			if perm.toString() == feature {
				return true
			}
		}
	}
	return false
}

func main() {
	http.HandleFunc("/users/", func(w http.ResponseWriter, r *http.Request) {
		paths := strings.Split(r.URL.Path, "/")
		var userId, productName string
		if len(paths) > 2 {
			userId = paths[2]
		}
		if len(paths) > 4 {
			productName = strings.Title(strings.Replace(paths[4], "-", " ", -1))
		}
		isUserDetailPage := r.Method == "GET" && userId != "" && productName == ""
		if isUserDetailPage {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/html")
			var user *UserEntity
			for idx, u := range users {
				if u.ID == userId {
					user = &users[idx]
					break
				}
			}
			var products []string
			for key, _ := range user.Roles {
				products = append(products, key)
			}
			t, err := template.ParseFiles("user_info.html")
			if err != nil {
				fmt.Errorf("error occurred when trying to parse the file %w", err)
				http.Redirect(w, r, "/page-not-found/", http.StatusFound)
				return
			}
			t.ExecuteTemplate(w, "user_info.html", struct {
				User struct {
					Id       string
					Products []string
				}
			}{
				User: struct {
					Id       string
					Products []string
				}{
					Id:       user.ID,
					Products: products,
				},
			})
		}
		isUserProductPage := r.Method == "GET" && userId != "" && productName != ""
		if isUserProductPage {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/html")
			var user *UserEntity
			for idx, u := range users {
				if u.ID == userId {
					user = &users[idx]
					break
				}
			}
			if user == nil {
				http.Redirect(w, r, "/page-not-found/", http.StatusFound)
			}
			t, err := template.ParseFiles("user_detail.html", "user_permissions.html")
			if err != nil {
				fmt.Errorf("error occurred when trying to parse the file %w", err)
				http.Redirect(w, r, "/page-not-found/", http.StatusFound)
				return
			}
			var userPermissions []string
			permissions, ok := user.Roles[productName]
			if !ok {
				http.Redirect(w, r, "/page-not-found/", http.StatusFound)
			}
			for _, p := range permissions {
				for _, action := range p {
					userPermissions = append(userPermissions, action.toString())
				}
			}
			style, ok := map[string]struct {
				Colors struct {
					ButtonBg string
					Button   string
				}
			}{
				"Product1": {
					Colors: struct {
						ButtonBg string
						Button   string
					}{
						ButtonBg: "yellow",
						Button:   "black",
					},
				},
				"Product2": {
					Colors: struct {
						ButtonBg string
						Button   string
					}{
						ButtonBg: "purple",
						Button:   "green",
					},
				},
			}[productName]
			if !ok {
				fmt.Errorf("style not found for product %s", productName)
				return
			}
			t.ExecuteTemplate(w, "user_detail.html", struct {
				Permissions []string
				ProductName string
				Style       struct {
					Colors struct {
						ButtonBg string
						Button   string
					}
				}
			}{
				Permissions: userPermissions,
				ProductName: productName,
				Style:       style,
			})
			return
		}
		if r.Method != "GET" {
			http.Redirect(w, r, "/page-not-found/", http.StatusFound)
		}
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Redirect(w, r, "/page-not-found/", http.StatusFound)
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html")
		t, err := template.ParseFiles("index.html", "user_table.html")
		if err != nil {
			fmt.Errorf("error occurred when trying to parse the file %w", err)
			http.Redirect(w, r, "/page-not-found/", http.StatusFound)
			return
		}
		t.ExecuteTemplate(w, "index.html", struct {
			Users []UserEntity
		}{
			Users: users,
		})
		return
	})
	http.HandleFunc("/page-not-found/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>Page not found</h1>")
	})
	fmt.Println("server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

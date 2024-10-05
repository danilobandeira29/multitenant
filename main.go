package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type SubscriptionType int

const (
	Basic SubscriptionType = iota
	Premium
)

func (s SubscriptionType) String() string {
	return [...]string{
		"Basic",
		"Premium",
	}[s]
}

type User struct {
	ID                   string
	Name                 string
	DeactivatedAt        *time.Time
	ProductSubscriptions map[string][]SubscriptionType
}

var users = []*User{
	{
		ID:            "1",
		Name:          "Danilo Bandeira",
		DeactivatedAt: nil,
		ProductSubscriptions: map[string][]SubscriptionType{
			"Product1": {
				Basic,
			},
		},
	},
	{
		ID:            "2",
		Name:          "Ana Banana",
		DeactivatedAt: nil,
		ProductSubscriptions: map[string][]SubscriptionType{
			"Product1": {
				Premium,
			},
			"Product2": {
				Premium,
			},
		},
	},
}

func (u *User) addSubscription(productName string, s SubscriptionType) error {
	subscriptions, ok := u.ProductSubscriptions[productName]
	if !ok {
		u.ProductSubscriptions[productName] = []SubscriptionType{s}
		return nil
	}
	var hasSub bool
	for _, subscriptionName := range subscriptions {
		if subscriptionName.String() == s.String() {
			hasSub = true
		}
	}
	if !hasSub {
		u.ProductSubscriptions[productName] = append(u.ProductSubscriptions[productName], []SubscriptionType{s}...)
	}
	return nil
}

func (u *User) subscriptionsFor(productName string) []SubscriptionType {
	subscriptions, ok := u.ProductSubscriptions[productName]
	if !ok {
		return make([]SubscriptionType, 0)
	}
	result := make([]SubscriptionType, len(subscriptions))
	copy(result, subscriptions)
	return result
}

func hasSubscription(entity User, feature []string, product string) bool {
	permissions, ok := entity.ProductSubscriptions[product]
	if !ok {
		return false
	}
	for _, p := range permissions {
		for _, f := range feature {
			if p.String() == f {
				return true
			}
		}
	}
	return false
}

func main() {
	mux := http.NewServeMux()
	logFile, err := os.OpenFile("logs.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		panic("not possible to create file 'logs.log'")
	}
	defer logFile.Close()
	multiWriter := io.MultiWriter(logFile, os.Stdout)
	log.SetOutput(multiWriter)
	mux.HandleFunc("/users/{userId}/", func(w http.ResponseWriter, r *http.Request) {
		userId := r.PathValue("userId")
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html")
		var user *User
		for idx, u := range users {
			if u.ID == userId {
				user = users[idx]
				break
			}
		}
		var products []string
		for key, _ := range user.ProductSubscriptions {
			products = append(products, key)
		}
		t, err := template.ParseFiles("user_info.html")
		if err != nil {
			log.Printf("error occurred when trying to parse the file %v", err)
			http.Redirect(w, r, "/page-not-found/", http.StatusFound)
			return
		}
		if err = t.ExecuteTemplate(w, "user_info.html", struct {
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
		}); err != nil {
			log.Printf("error when executing template %v", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	})
	mux.HandleFunc("/users/{userId}/products/{productId}/", func(w http.ResponseWriter, r *http.Request) {
		userId := r.PathValue("userId")
		productName := strings.Title(strings.Replace(r.PathValue("productId"), "-", " ", -1))
		if r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "text/html")
			var user *User
			for idx, u := range users {
				if u.ID == userId {
					user = users[idx]
					break
				}
			}
			if user == nil {
				http.Error(w, "user not found", http.StatusNotFound)
				return
			}
			t, err := template.ParseFiles("user_detail.html", "user_permissions.html")
			if err != nil {
				log.Printf("error occurred when trying to parse the file %v\n", err)
				http.Redirect(w, r, "/page-not-found/", http.StatusFound)
				return
			}
			var userPermissions []string
			permissions, ok := user.ProductSubscriptions[productName]
			if !ok {
				http.Redirect(w, r, "/page-not-found/", http.StatusFound)
			}
			for _, p := range permissions {
				userPermissions = append(userPermissions, p.String())
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
				log.Printf("style not found for product %s\n", productName)
				http.Error(w, "product not found", http.StatusNotFound)
				return
			}
			if err = t.ExecuteTemplate(w, "user_detail.html", struct {
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
			}); err != nil {
				log.Printf("error when executing template %v", err)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
			return
		}
		if r.Method == http.MethodPatch {
			var user *User
			for _, u := range users {
				if u.ID == userId {
					user = u
					break
				}
			}
			var dto struct {
				Type string `json:"type"`
			}
			err := json.NewDecoder(r.Body).Decode(&dto)
			defer func(body io.ReadCloser) {
				if err = body.Close(); err != nil {
					log.Printf("error when closing body %v", err)
				}
			}(r.Body)
			if err != nil {
				log.Printf("error in body. must me a json %v", err)
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}
			var newSubscription SubscriptionType
			for _, key := range []SubscriptionType{
				Premium,
				Basic,
			} {
				if key.String() == dto.Type {
					newSubscription = key
				}
			}
			if err = user.addSubscription(productName, newSubscription); err != nil {
				log.Printf("error when trying to add subscription %s for product %s to user with id %s\n", newSubscription.String(), productName, userId)
				http.Error(w, fmt.Sprintf("not possivel to add subscription for this user: %v", err), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			resp := map[string]interface{}{
				"data": map[string]interface{}{
					"message": fmt.Sprintf("user's permissions %v", user.ProductSubscriptions[productName]),
				},
				"kind": "success",
			}
			if err = json.NewEncoder(w).Encode(resp); err != nil {
				log.Printf("error when trying to send response to the client")
			}
			return
		}
	})
	mux.HandleFunc("/users/{userId}/products/{productId}/videos/", func(w http.ResponseWriter, r *http.Request) {
		userId := r.PathValue("userId")
		productName := strings.Title(strings.Replace(r.PathValue("productId"), "-", " ", -1))
		var videos []struct {
			Type        string `json:"type"`
			Url         string `json:"url"`
			Thumbnail   string `json:"thumbnail"`
			Title       string `json:"title"`
			ProductName string `json:"product_name"`
		}
		bytes, err := os.ReadFile("videos_" + productName + ".json")
		if err != nil {
			resp := map[string]interface{}{
				"kind": "error",
				"error": map[string]interface{}{
					"message": err.Error(),
				},
			}
			jsonResp, err := json.Marshal(resp)
			if err != nil {
				log.Printf("error when trying to parse resp json %v\n", err)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			if _, err = w.Write(jsonResp); err != nil {
				log.Printf(fmt.Sprintf("%s: not possible to send responde to the client", r.URL))
			}
			return
		}
		err = json.Unmarshal(bytes, &videos)
		if err != nil {
			log.Printf("error occurred when trying to unmarshal json %v\n", err)
			http.Error(w, fmt.Sprintf("error when trying to unmarshal json file %v\n", err), http.StatusBadRequest)
			return
		}
		var user *User
		for _, u := range users {
			if u.ID == userId {
				user = u
				break
			}
		}
		if !hasSubscription(*user, []string{"Basic", "Premium"}, productName) {
			http.Redirect(w, r, "/forbidden/", http.StatusFound)
			return
		}
		var availableVideos []struct {
			Type        string `json:"type"`
			Url         string `json:"url"`
			Thumbnail   string `json:"thumbnail"`
			Title       string `json:"title"`
			ProductName string `json:"product_name"`
		}
		if !hasSubscription(*user, []string{"Premium"}, productName) {
			for _, v := range videos {
				if v.Type == "basic" {
					availableVideos = append(availableVideos, v)
				}
			}
		} else {
			availableVideos = videos
		}
		t, err := template.ParseFiles("user_videos.html")
		if err != nil {
			fmt.Errorf("error occurred when trying to parse the file %w", err)
			http.Redirect(w, r, "/page-not-found/", http.StatusFound)
			return
		}
		t.ExecuteTemplate(w, "user_videos.html", availableVideos)
		return
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
			Users []*User
		}{
			Users: users,
		})
		return
	})
	mux.HandleFunc("/page-not-found/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>Page not found</h1>")
	})
	mux.HandleFunc("/forbidden/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "<h1>Forbidden</h1>")
	})
	fmt.Println("server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}

/**
1. [ ] Criar botao no front para adicionar permissao para o user
2. [ ] Aplicar lib de estilização
3. [ ] Testes automatizados?
*/

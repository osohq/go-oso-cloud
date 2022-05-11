package main

import (
	"fmt"
	"log"

	oso "github.com/osohq/go-oso"
)

type User struct {
	id int
}

func (u User) Id() string {
	return fmt.Sprint(u.id)
}

func (u User) Type() string {
	return "User"
}

type Repo struct {
	id int
}

func (r Repo) Id() string {
	return fmt.Sprint(r.id)
}

func (r Repo) Type() string {
	return "Repo"
}

func main() {
	oso := oso.NewClient("http://localhost:8080", "dF8wMTIzNDU2Nzg5Om9zb190ZXN0X3Rva2Vu")
	oso.Policy(`
		actor User {}

		resource Repo {
			roles = ["member"];
			permissions = ["read"];
			relations = { parent: Repo };
		}
	`)
	allowed, e := oso.Authorize(User{id: 1}, "read", Repo{id: 2})
	if e != nil || allowed != false {
		log.Fatalln(e, "Authorize", allowed)
	}

	results, e := oso.List(User{id: 1}, "read", Repo{})
	if e != nil || len(results) != 0 {
		log.Fatalln(e)
	}

	e = oso.Tell("has_relation", Repo{id: 2}, oso.String("parent"), Repo{id: 3})
	if e != nil {
		log.Fatalln(e)
	}

	e = oso.Delete("has_relation", Repo{id: 2}, oso.String("parent"), Repo{id: 3})
	if e != nil {
		log.Fatalln(e)
	}

	e = oso.Tell("has_role", Repo{id: 2}, oso.String("member"), User{id: 1})
	if e != nil {
		log.Fatalln(e)
	}

	roles, e := oso.Get("has_role", Repo{id: 2}, oso.String("member"), User{id: 1})
	if e != nil || len(roles) != 1 || roles[0].Predicate != "has_role" {
		log.Fatalln(e)
	}

	e = oso.Delete("has_role", Repo{id: 2}, oso.String("member"), User{id: 1})
	if e != nil {
		log.Fatalln(e)
	}
	log.Printf("Success")
}

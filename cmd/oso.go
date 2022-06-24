package main

import (
	"fmt"
	"log"

	oso "github.com/osohq/go-oso-cloud"
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
	o := oso.NewClient("http://localhost:8080", "dF8wMTIzNDU2Nzg5Om9zb190ZXN0X3Rva2Vu")
	o.Policy(`
		actor User {}

		resource Repo {
			roles = ["member"];
			permissions = ["read"];
			relations = { parent: Repo };
      		"read" if "member";
		}
	`)
	allowed, e := o.Authorize(User{id: 1}, "read", Repo{id: 2})
	if e != nil || allowed != false {
		log.Fatalln(e, "Authorize", allowed)
	}

	results, e := o.List(User{id: 1}, "read", Repo{})
	if e != nil || len(results) != 0 {
		log.Fatalln(e)
	}

	e = o.Tell("has_relation", Repo{id: 2}, oso.String("parent"), Repo{id: 2})
	if e != nil {
		log.Fatalln(e)
	}

	e = o.Delete("has_relation", Repo{id: 3}, oso.String("parent"), Repo{id: 2})
	if e != nil {
		log.Fatalln(e)
	}

	e = o.Tell("has_role", User{id: 1}, oso.String("member"), Repo{id: 2})
	if e != nil {
		log.Fatalln(e)
	}

	roles, e := o.Get("has_role", User{id: 1}, oso.String("member"), Repo{id: 2})
	if e != nil || len(roles) != 1 || roles[0].Predicate != "has_role" {
		log.Fatalln(e)
	}

	allowed_again, e := o.Authorize(User{id: 1}, "read", Repo{id: 2})
	if e != nil || allowed_again != true {
		log.Fatalln(e, "Authorize", allowed_again)
	}

	e = o.Delete("has_role", User{id: 1}, oso.String("member"), Repo{id: 2})
	if e != nil {
		log.Fatalln(e)
	}

	log.Printf("Success")
}

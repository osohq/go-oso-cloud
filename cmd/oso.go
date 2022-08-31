package main

import (
	"fmt"
	"log"

	oso "github.com/osohq/go-oso-cloud"
)

type User struct {
	id int
}

func (u User) Instance() oso.Instance {
	typ, id := "User", fmt.Sprint(u.id)
	return oso.Instance{Type: typ, Id: id}
}

type Repo struct {
	id int
}

func (r Repo) Instance() oso.Instance {
	typ, id := "Repo", fmt.Sprint(r.id)
	return oso.Instance{Type: typ, Id: id}
}

func main() {
	o := oso.NewClient("http://localhost:8081", "e_0123456789_12345_osotesttoken01xiIn")
	o.Policy(`
		actor User {}

		resource Repo {
			roles = ["member"];
			permissions = ["read"];
			relations = { parent: Repo };
      		"read" if "member";
		}
	`)
	allowed, e := o.Authorize(User{id: 1}.Instance(), "read", Repo{id: 2}.Instance())
	if e != nil || allowed {
		log.Fatalln(e, "Authorize", allowed)
	}

	results, e := o.List(User{id: 1}.Instance(), "read", "Repo", nil)
	if e != nil || len(results) != 0 {
		log.Fatalln(e)
	}

	e = o.Tell("has_relation", Repo{id: 2}.Instance(), oso.String("parent"), Repo{id: 2}.Instance())
	if e != nil {
		log.Fatalln(e)
	}

	e = o.Delete("has_relation", Repo{id: 3}.Instance(), oso.String("parent"), Repo{id: 2}.Instance())
	if e != nil {
		log.Fatalln(e)
	}

	e = o.Tell("has_role", User{id: 1}.Instance(), oso.String("member"), Repo{id: 2}.Instance())
	if e != nil {
		log.Fatalln(e)
	}

	roles, e := o.Get("has_role", User{id: 1}.Instance(), oso.String("member"), Repo{id: 2}.Instance())
	if e != nil || len(roles) != 1 || roles[0].Predicate != "has_role" {
		log.Fatalln(e)
	}

	allowed_again, e := o.Authorize(User{id: 1}.Instance(), "read", Repo{id: 2}.Instance())
	if e != nil || !allowed_again {
		log.Fatalln(e, "Authorize", allowed_again)
	}

	e = o.Delete("has_role", User{id: 1}.Instance(), oso.String("member"), Repo{id: 2}.Instance())
	if e != nil {
		log.Fatalln(e)
	}

	log.Printf("Success")
}

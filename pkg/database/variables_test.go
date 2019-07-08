package database

import (
	"math"

	"github.com/go-xorm/builder"
)

type testEnv struct {
	table string

	empty, fail, get, getExpected, select1, select2, select3, select4, select5,
	selectResult, selectExpected, create, updateBefore, updateAfter, delete,
	execExpected, commit, rollback interface{}

	exec builder.Eq

	selectOrder     string
	selectCondition string
	selectArgs      []interface{}
}

func usersTestEnv() *testEnv {
	env := &testEnv{}
	env.table = (&User{}).TableName()
	env.empty = &User{}
	env.fail = &User{
		ID: math.MaxUint64,
	}
	env.getExpected = &User{
		Login:    "get",
		Password: []byte("get_password"),
	}
	env.get = &User{
		Login: env.getExpected.(*User).Login,
	}
	env.select1 = &User{
		Login:    "select1",
		Password: []byte("select_password"),
	}
	env.select2 = &User{
		Login:    "select2",
		Password: env.select1.(*User).Password,
	}
	env.select3 = &User{
		Login:    "select3",
		Password: []byte("not_select_password"),
	}
	env.select4 = &User{
		Login:    "select4",
		Password: env.select1.(*User).Password,
	}
	env.select5 = &User{
		Login:    "select5",
		Password: env.select1.(*User).Password,
	}
	env.selectResult = &[]*User{}
	env.selectExpected = &[]*User{env.select4.(*User), env.select2.(*User)}
	env.selectOrder = "login DESC"
	env.selectCondition = "password = ?"
	env.selectArgs = []interface{}{env.select1.(*User).Password}
	env.create = &User{
		Login:    "create",
		Password: []byte("create_password"),
	}
	env.updateBefore = &User{
		Login:    "update",
		Password: []byte("update_password"),
	}
	env.updateAfter = &User{
		Login:    "updated",
		Password: []byte("updated_password"),
	}
	env.delete = &User{
		Login:    "delete",
		Password: []byte("delete_password"),
	}
	env.execExpected = &User{
		Login:    "execute",
		Password: []byte("execute_password"),
	}
	env.exec = builder.Eq{
		"login":    env.execExpected.(*User).Login,
		"password": env.execExpected.(*User).Password,
	}
	env.commit = &User{
		Login:    "commit",
		Password: []byte("commit_password"),
	}
	env.rollback = &User{
		Login:    "rollback",
		Password: []byte("rollbackpassword"),
	}

	return env
}

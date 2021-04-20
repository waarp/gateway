package types

import "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tk/utils"

type CypherText string

func (c *CypherText) FromDB(bytes []byte) error {
	if len(bytes) == 0 {
		return nil
	}
	plain, err := utils.AESDecrypt(string(bytes))
	if err != nil {
		return err
	}
	*c = CypherText(plain)
	return nil
}

func (c *CypherText) ToDB() ([]byte, error) {
	if *c == "" {
		return nil, nil
	}
	cypher, err := utils.AESCrypt(string(*c))
	if err != nil {
		return nil, err
	}
	return []byte(cypher), nil
}

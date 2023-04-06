package migrations

import (
	"encoding/json"
	"fmt"
)

type ver0_7_5SplitR66TLS struct{}

type ver0_7_5SplitR66TLSRow struct {
	id    int64
	proto string
	conf  map[string]any
}

func (ver0_7_5SplitR66TLS) getR66AgentsList(db Actions, tbl string) ([]*ver0_7_5SplitR66TLSRow, error) {
	rows, err := db.Query(`SELECT id,protocol,proto_config FROM ` + tbl +
		` WHERE protocol = 'r66' OR protocol = 'r66-tls'`)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve the R66 %s: %w", tbl, err)
	}

	defer rows.Close() //nolint:errcheck //error is irrelevant here

	var res []*ver0_7_5SplitR66TLSRow

	for rows.Next() {
		var (
			row  ver0_7_5SplitR66TLSRow
			conf string
		)

		if err := rows.Scan(&row.id, &row.proto, &conf); err != nil {
			return nil, fmt.Errorf("failed to parse %s values: %w", tbl, err)
		}

		if err := json.Unmarshal([]byte(conf), &row.conf); err != nil {
			return nil, fmt.Errorf("failed to parse the %s's proto config: %w", tbl, err)
		}

		res = append(res, &row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error while iterating over the %s values: %w", tbl, err)
	}

	return res, nil
}

func (v ver0_7_5SplitR66TLS) Up(db Actions) error {
	for _, tbl := range []string{"local_agents", "remote_agents"} {
		r66Agents, getErr := v.getR66AgentsList(db, tbl)
		if getErr != nil {
			return getErr
		}

		for _, agent := range r66Agents {
			if anyIsTLS, hasTLS := agent.conf["isTLS"]; hasTLS {
				if isTLS, isBool := anyIsTLS.(bool); isBool && isTLS {
					agent.proto = "r66-tls"

					if err := db.Exec(`UPDATE `+tbl+` SET protocol=? WHERE id=?`,
						agent.proto, agent.id); err != nil {
						return fmt.Errorf("failed to update the %s's protocol information: %w", tbl, err)
					}
				}
			}
		}
	}

	return nil
}

func (v ver0_7_5SplitR66TLS) Down(db Actions) error {
	for _, tbl := range []string{"local_agents", "remote_agents"} {
		r66Agents, getErr := v.getR66AgentsList(db, tbl)
		if getErr != nil {
			return getErr
		}

		for _, agent := range r66Agents {
			if _, hasTLS := agent.conf["isTLS"]; !hasTLS {
				agent.conf["isTLS"] = agent.proto == "r66-tls"

				conf, err := json.Marshal(agent.conf)
				if err != nil {
					return fmt.Errorf("failed to marshal the r66 proto config: %w", err)
				}

				if err := db.Exec(`UPDATE `+tbl+` SET proto_config=? WHERE id=?`,
					string(conf), agent.id); err != nil {
					return fmt.Errorf("failed to update the %s's r66 proto config: %w", tbl, err)
				}
			}
		}
	}

	if err := db.Exec(`UPDATE local_agents SET protocol='r66' WHERE protocol='r66-tls'`); err != nil {
		return fmt.Errorf("failed to update the local agent's protocol information: %w", err)
	}

	if err := db.Exec(`UPDATE remote_agents SET protocol='r66' WHERE protocol='r66-tls'`); err != nil {
		return fmt.Errorf("failed to update the remote agent's protocol information: %w", err)
	}

	return nil
}

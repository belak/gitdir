package gitdir

/*
// AcceptInvite attempts to accept an invite and set up the given user.
func (serv *Server) AcceptInvite(invite string, key PublicKey) bool { //nolint:funlen
	serv.lock.Lock()
	defer serv.lock.Unlock()

	// It's expensive to lock, so we need to do the fast stuff first and bail as
	// early as possible if it's not valid.

	// Step 1: Look up the invite
	username, ok := serv.settings.Invites[invite]
	if !ok {
		return false
	}

	fmt.Println("found user")

	adminRepo, err := EnsureRepo("admin/admin", true)
	if err != nil {
		log.Warn().Err(err).Msg("Admin repo doesn't exist")
		return false
	}

	err = adminRepo.UpdateFile("config.yml", func(data []byte) ([]byte, error) {
		rootNode, _, err := ensureSampleConfigYaml(data) //nolint:govet
		if err != nil {
			return nil, err
		}

		// We can assume the config file is in a valid format because of
		// ensureSampleConfig
		targetNode := rootNode.Content[0]

		// Step 2: Ensure the user exists and is not disabled.
		usersVal := yamlLookupVal(targetNode, "users")
		userVal, _ := yamlEnsureKey(usersVal, username, &yaml.Node{Kind: yaml.MappingNode}, "", false)
		_ = yamlRemoveKey(userVal, "disabled")

		// Step 3: Add the key to the user
		keysVal, _ := yamlEnsureKey(userVal, "keys", &yaml.Node{Kind: yaml.SequenceNode}, "", false)
		keysVal.Content = append(keysVal.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: key.MarshalAuthorizedKey(),
		})

		// Step 4: Remove the invite (and any others this user owns)
		var staleInvites []string
		invitesVal := yamlLookupVal(targetNode, "invites")

		for i := 0; i+1 < len(invitesVal.Content); i += 2 {
			if invitesVal.Content[i+1].Value == username {
				staleInvites = append(staleInvites, invitesVal.Content[i].Value)
			}
		}

		for _, val := range staleInvites {
			yamlRemoveKey(invitesVal, val)
		}

		// Step 5: Re-encode back to yaml
		data, err = yamlEncode(rootNode)
		return data, err
	})
	if err != nil {
		log.Warn().Err(err).Msg("Failed to update config")
		return false
	}

	err = adminRepo.Commit("Added "+username+" from invite "+invite, nil)
	if err != nil {
		return false
	}

	err = serv.reloadInternal()

	// The invite was successfully accepted if the server reloaded properly.
	return err == nil
}
*/

package gitdir

import (
	"errors"
	"fmt"
	"strings"

	"github.com/belak/go-gitdir/models"
)

// Validate will ensure the config is valid and return any errors.
func (c *Config) Validate(user *UserSession, pk *models.PublicKey) error {
	return newMultiError(
		c.validateUser(user),
		c.validatePublicKey(pk),
		c.validateAdmins(),
		c.validateGroupLoop(),
	)
}

func (c *Config) validateUser(u *UserSession) error {
	if _, ok := c.adminConfig.Users[u.Username]; !ok {
		return fmt.Errorf("cannot remove current user: %s", u.Username)
	}

	return nil
}

func (c *Config) validatePublicKey(pk *models.PublicKey) error {
	if _, ok := c.publicKeys[pk.RawMarshalAuthorizedKey()]; !ok {
		return fmt.Errorf("cannot remove current private key: %s", pk.MarshalAuthorizedKey())
	}

	return nil
}

func (c *Config) validateAdmins() error {
	for _, user := range c.adminConfig.Users {
		if user.IsAdmin {
			return nil
		}
	}

	return errors.New("no admins defined")
}

func (c *Config) validateGroupLoop() error {
	var errors []error

	// Essentially this is "do a tree traversal on the groups"
	for groupName := range c.adminConfig.Groups {
		errors = append(errors, c.validateGroupLoopInternal(groupName, nil))
	}

	return newMultiError(errors...)
}

func (c *Config) validateGroupLoopInternal(groupName string, groupPath []string) error {
	// If we hit a group loop, return the path to get here
	if listContainsStr(groupPath, groupName) {
		return fmt.Errorf("group loop found: %s", strings.Join(append(groupPath, groupName), ", "))
	}

	groupPath = append(groupPath, groupName)

	for _, lookup := range c.adminConfig.Groups[groupName] {
		if strings.HasPrefix(lookup, groupPrefix) {
			intGroupName := strings.TrimPrefix(lookup, groupPrefix)

			if err := c.validateGroupLoopInternal(intGroupName, groupPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// Copyright 2017 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package git

import (
	"context"
	"testing"

	"github.com/keybase/client/go/externals"
	"github.com/keybase/client/go/kbfs"
	"github.com/keybase/client/go/kbtest"
	"github.com/keybase/client/go/libkb"
	"github.com/keybase/client/go/protocol/keybase1"
	"github.com/keybase/client/go/teams"
	"github.com/stretchr/testify/require"
)

// Copied from the teams tests.
func SetupTest(tb testing.TB, name string, depth int) (tc libkb.TestContext) {
	tc = libkb.SetupTest(tb, name, depth+1)
	tc.G.SetServices(externals.GetServices())
	teams.NewTeamLoaderAndInstall(tc.G)
	return tc
}

func doPut(t *testing.T, g *libkb.GlobalContext, teamName string, repoID string, repoName string) {
	err := PutMetadata(context.Background(), g, keybase1.PutGitMetadataArg{
		Folder: keybase1.Folder{
			Name:       teamName,
			Private:    true,
			FolderType: keybase1.FolderType_TEAM,
		},
		RepoID: keybase1.RepoID(repoID),
		Metadata: keybase1.GitLocalMetadata{
			RepoName: keybase1.GitRepoName(repoName),
		},
	})
	require.NoError(t, err)
}

func TestPutAndGet(t *testing.T) {
	tc := SetupTest(t, "team", 1)
	defer tc.Cleanup()

	// Note that the length limit for a team name, with the additional suffix
	// below, is 16 characters. We have 5 to play with, including the implicit
	// underscore after the prefix.
	u, err := kbtest.CreateAndSignupFakeUser("t", tc.G)
	require.NoError(t, err)

	// Create two teams, so that we can test filtering by TeamID.
	teamName1 := u.Username + "t1"
	err = teams.CreateRootTeam(context.Background(), tc.G, teamName1)
	require.NoError(t, err)
	team1, err := tc.G.GetTeamLoader().Load(context.Background(), keybase1.LoadTeamArg{Name: teamName1})
	require.NoError(t, err)

	teamName2 := u.Username + "t2"
	err = teams.CreateRootTeam(context.Background(), tc.G, teamName2)
	require.NoError(t, err)

	// Create two git repos, one in each team. Remember that all we're
	// "creating" here is metadata.
	doPut(t, tc.G, teamName1, "abc123", "repoNameFirst")
	doPut(t, tc.G, teamName2, "def456", "repoNameSecond")
	expectedIDNames := map[string]string{
		"abc123": "repoNameFirst",
		"def456": "repoNameSecond",
	}

	// Get all repos, and make sure both come back.
	allRepos, err := GetAllMetadata(context.Background(), tc.G)
	require.NoError(t, err)
	require.Equal(t, 2, len(allRepos), "expected to get both repos back, found: %d", len(allRepos))
	for _, repo := range allRepos {
		require.Equal(t, expectedIDNames[string(repo.RepoID)], string(repo.LocalMetadata.RepoName))
		require.Equal(t, repo.Folder.FolderType, keybase1.FolderType_TEAM)
		require.Equal(t, repo.Folder.Private, true)
		require.Equal(t, repo.ServerMetadata.LastModifyingUsername, u.Username)
		require.Equal(t, repo.CanDelete, true)
	}

	// Now get the repos for just one team. Should be only one of the two we just created.
	oneRepoList, err := GetMetadata(context.Background(), tc.G, keybase1.Folder{
		Name:       teamName1,
		Private:    true,
		FolderType: keybase1.FolderType_TEAM,
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(oneRepoList), "expected to get only one repo back, found: %d", len(oneRepoList))
	oneRepo := oneRepoList[0]
	require.Equal(t, "repoNameFirst", string(oneRepo.LocalMetadata.RepoName))
	require.Equal(t, kbtest.DefaultDeviceName, oneRepo.ServerMetadata.LastModifyingDeviceName)
	require.Equal(t, string(team1.Chain.Id+"_abc123"), oneRepo.GlobalUniqueID)
	require.Equal(t, "keybase://team/"+teamName1+"/repoNameFirst", oneRepo.RepoUrl)
	require.Equal(t, oneRepo.CanDelete, true)
}

func TestDeleteRepo(t *testing.T) {
	tc := SetupTest(t, "team", 1)
	defer tc.Cleanup()

	// Note that the length limit for a team name, with the additional suffix
	// below, is 16 characters. We have 5 to play with, including the implicit
	// underscore after the prefix.
	u, err := kbtest.CreateAndSignupFakeUser("t", tc.G)
	require.NoError(t, err)

	// Create a personal repo for this user. (This also exercises the
	// personal-repos-create-an-implicit-team flow.)
	repoName := keybase1.GitRepoName("testRepo123")
	folder := keybase1.Folder{
		Name:       u.Username,
		Private:    true,
		FolderType: keybase1.FolderType_PRIVATE,
	}
	err = PutMetadata(context.Background(), tc.G, keybase1.PutGitMetadataArg{
		Folder: folder,
		RepoID: keybase1.RepoID("abc123"),
		Metadata: keybase1.GitLocalMetadata{
			RepoName: repoName,
		},
	})
	require.NoError(t, err)

	// Get all repos, and make sure we see that one.
	allRepos, err := GetAllMetadata(context.Background(), tc.G)
	require.NoError(t, err)
	require.Equal(t, 1, len(allRepos), "expected to see 1 git repo, saw %d", len(allRepos))
	require.Equal(t, repoName, allRepos[0].LocalMetadata.RepoName)

	// Now delete that repo.
	err = DeleteMetadata(context.Background(), tc.G, folder, repoName)
	require.NoError(t, err)

	// Finally, get all repos again, and make sure it's gone.
	allRepos, err = GetAllMetadata(context.Background(), tc.G)
	require.NoError(t, err)
	require.Equal(t, 0, len(allRepos), "expected the repo to get deleted")
}

func TestPutAndGetImplicitTeam(t *testing.T) {
	tc := SetupTest(t, "team", 1)
	defer tc.Cleanup()

	u1, err := kbtest.CreateAndSignupFakeUser("t", tc.G)
	require.NoError(t, err)
	u2, err := kbtest.CreateAndSignupFakeUser("t", tc.G)
	require.NoError(t, err)

	// Create two repos, one in u2's personal directory (since we're signed in
	// as u2 at this point), and one in the directory that u1 and u2 share. One
	// of the things we'll be testing is that the second one does *not* show up
	// in the results of GetAllMetadata. This is a product choice -- we want to
	// pretent they kinda don't exist.

	repoName1 := keybase1.GitRepoName("me only repo")
	testFolder1 := keybase1.Folder{
		Name:       u2.Username,
		Private:    true,
		FolderType: keybase1.FolderType_PRIVATE,
	}
	err = PutMetadata(context.TODO(), tc.G, keybase1.PutGitMetadataArg{
		Folder: testFolder1,
		RepoID: keybase1.RepoID("abc123"),
		Metadata: keybase1.GitLocalMetadata{
			RepoName: repoName1,
		},
	})
	require.NoError(t, err)

	repoName2 := keybase1.GitRepoName("two person private repo")
	normalizedTLFName, err := kbfs.NormalizeNamesInTLF([]string{u1.Username, u2.Username}, nil, "")
	require.NoError(t, err)
	testFolder2 := keybase1.Folder{
		Name:       normalizedTLFName,
		Private:    true,
		FolderType: keybase1.FolderType_PRIVATE,
	}
	err = PutMetadata(context.Background(), tc.G, keybase1.PutGitMetadataArg{
		Folder: testFolder2,
		RepoID: keybase1.RepoID("abc123"),
		Metadata: keybase1.GitLocalMetadata{
			RepoName: repoName2,
		},
	})
	require.NoError(t, err)

	// Now make sure we can query these repos (or not, as appropriate for the
	// multi-person-private case).

	assertStuffAboutRepo := func(t *testing.T, repo keybase1.GitRepoResult, folder keybase1.Folder, repoName keybase1.GitRepoName) {
		require.Equal(t, repoName, repo.LocalMetadata.RepoName)
		require.Equal(t, keybase1.FolderType_PRIVATE, repo.Folder.FolderType)
		require.Equal(t, true, repo.Folder.Private)
		require.Equal(t, kbtest.DefaultDeviceName, repo.ServerMetadata.LastModifyingDeviceName)
		require.Equal(t, "keybase://private/"+folder.Name+"/"+string(repoName), repo.RepoUrl)
		require.Equal(t, repo.CanDelete, true)
	}

	// Test fetching the private repo with GetMetadata.
	oneRepo, err := GetMetadata(context.Background(), tc.G, testFolder1)
	require.NoError(t, err)
	require.Equal(t, 1, len(oneRepo))
	assertStuffAboutRepo(t, oneRepo[0], testFolder1, repoName1)

	// Also test fetching the 2-person private repo with GetMetadata. This
	// *should* work, even though it's hidden from GetAllMetadata.
	oneRepo, err = GetMetadata(context.Background(), tc.G, testFolder2)
	require.NoError(t, err)
	require.Equal(t, 1, len(oneRepo))
	assertStuffAboutRepo(t, oneRepo[0], testFolder2, repoName2)

	// Finally test the GetAllMetadata results. This should *not* see the
	// 2-person repo.
	allRepos, err := GetAllMetadata(context.Background(), tc.G)
	require.NoError(t, err)
	require.Equal(t, 1, len(allRepos), "the two-person repo should be hidden!")
	assertStuffAboutRepo(t, allRepos[0], testFolder1, repoName1)
}

func TestPutAndGetWritersCantDelete(t *testing.T) {
	tc := SetupTest(t, "team", 1)
	defer tc.Cleanup()

	u1, err := kbtest.CreateAndSignupFakeUser("t", tc.G)
	require.NoError(t, err)
	u2, err := kbtest.CreateAndSignupFakeUser("t", tc.G)
	require.NoError(t, err)

	// u2 creates a team where u1 is just a writer
	teamName := u2.Username + "t1"
	err = teams.CreateRootTeam(context.Background(), tc.G, teamName)
	require.NoError(t, err)
	_, err = teams.AddMember(context.Background(), tc.G, teamName, u1.Username, keybase1.TeamRole_WRITER)
	if err != nil {
		t.Fatal(err)
	}

	// Create a git repo for that team
	doPut(t, tc.G, teamName, "abc123", "dummyRepoName")

	// Load the repo and confirm that u2 sees it as CanDelete=true.
	repos, err := GetAllMetadata(context.Background(), tc.G)
	require.NoError(t, err)
	require.Equal(t, 1, len(repos), "expected exactly one repo")
	require.Equal(t, true, repos[0].CanDelete, "owners/admins should be able to delete")

	// Now log in as u1, load the repo again, and confirm that u1 sees it as CanDelete=FALSE.
	err = tc.G.Logout()
	require.NoError(t, err)
	err = u1.Login(tc.G)
	require.NoError(t, err)
	repos2, err := GetAllMetadata(context.Background(), tc.G)
	require.NoError(t, err)
	require.Equal(t, 1, len(repos2), "expected exactly one repo")
	require.Equal(t, false, repos2[0].CanDelete, "non-admins must not be able to delete")
}
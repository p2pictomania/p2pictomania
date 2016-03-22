# p2pictomania

#### Contribution Guidelines
Lets follow an approach of submitting pull requests for new code changes. So that at least one other person gets to review the code before it can be merged.

Just to iterate over the steps to do before starting on a new feature:
- Locally create a new branch
```bash
git checkout master
# make sure you are at the latest version of master before you start
git checkout -b feature1
```
- Implement your feature and commit often (locally)
- Once your done with your feature and have committed
```bash
git checkout master
git pull origin master # get latest master again
git checkout feature1 # go back to your branch
git rebase master # rebase from master so your branch is updated. resolve conflicts if any.
```
- Squash all your commits for this feature into one commit. So it is easy to review the commit once pushed.
```bash
# if you made 5 commits in your feature squash the 5 latest commits into 1.
git rebase -i HEAD~5
#replace "pick" on the second and subsequent commits with "squash". Then save the commit.
```
- Push branch to github
```bash
git push origin feature1
```
- Now go to github and click the "create a new pull request" button on the project page and submit it.
- Code will be merged by anyone `other than` the person who committed it.

#### Dev Notes

- **03/21/2016** :
Set up the basic structure of the application. The app can be built by running
```bash
go build
./p2pictomania
```
The Web UI starts on port `8000` and the bootstrap server starts on port `5000`.

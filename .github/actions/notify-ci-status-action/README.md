# Notify CI status action

This github action is used to notify users in slack about a CI failure

## Developing

- Make changes to the code as needed.
- Make sure you run `npm install` and that a `node_modules` directory exists.
- `npm run build` to compile changes. It used `ncc` as [github recommends]((https://docs.github.com/en/actions/creating-actions/creating-a-javascript-action#commit-tag-and-push-your-action-to-github)) us to not merge the `node_modules`.
- Commit all the changes but make sure that `node_modules` is not accidentally commited.

## Testing

To test this action make sure to update the job in [../../workflows/reusable-notify-ci-status.yml](../../workflows/reusable-notify-ci-status.yml) to point to the development branch. And then also make sure to point it back to master after testing

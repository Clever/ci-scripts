After any changes to the actions `index.js` file, re-compile the index.js file with: `npm run build`

- this pulls in the `node_modules` ahead of time to prevent version/system issues
- according to [Github docs](https://docs.github.com/en/actions/creating-actions/creating-a-javascript-action#commit-tag-and-push-your-action-to-github): `Checking in your node_modules directory can cause problems. As an alternative, you can use a tool called @vercel/ncc to compile your code and modules into one file used for distribution.`

Commit the updated `index.js` and the `dist/index.ts`.

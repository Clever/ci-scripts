const core = require("@actions/core");
const axios = require("axios");
const { WebClient: SlackClient } = require("@slack/web-api");

const colorFailure = "#FF0000"; // red
const colorSuccess = "#00FF00"; // green

async function runAction() {
  const branch = core.getInput("branch");
  const description = core.getInput("description");
  const githubUsername = core.getInput("githubUsername");
  const repo = core.getInput("repo");
  const failedStep = core.getInput("failedStep") || "";
  const state = core.getInput("state");
  const targetURL = core.getInput("targetURL");

  let slackID = "";
  try {
    slackID = await getSlackIDFromEmail(githubUsername);
  } catch (error) {
    core.setFailed("Error retrieving slackID: " + error.message);
    return;
  }

  const message = formMessage(repo, branch, state, targetURL, failedStep);
  const attachments = formAttachments(description, failedStep);

  const slackClient = new SlackClient(process.env.SLACK_BOT_TOKEN);
  let resp;
  try {
    resp = await slackClient.chat.postMessage({
      channel: `@${slackID}`,
      text: message,
      attachments,
    });
  } catch (error) {
    core.setFailed("Error sending message via Slack Client: " + error.message);
    return;
  }
  core.setOutput("messageTS", resp.message.ts);
}

async function getSlackIDFromEmail(githubUsername) {
  const url = `${process.env.CIRCLE_CI_INTEGRATIONS_URL}/users/slackID?githubUsername=${githubUsername}`;
  const options = {
    auth: {
      username: process.env.CIRCLE_CI_INTEGRATIONS_USERNAME,
      password: process.env.CIRCLE_CI_INTEGRATIONS_PASSWORD,
    },
  };
  try {
    const resp = await axios.get(url, options);
    if (!resp.data.slackID) {
      // Shouldn't reach here as circle-ci-integrations should return a 404
      throw new Error("Empty SlackID found for github user");
    }
    return resp.data.slackID;
  } catch (err) {
    if (err.response && err.response.status == "404") {
      throw new Error("SlackID not found for github user");
    }
    throw err;
  }
}

function formMessage(repo, branch, state, targetURL, failedStep) {
  // if targetUrl is not set and failed step contains circleCI then set link to circleCi
  if (!targetURL && failedStep.toLowerCase().includes("circleci")) {
    targetURL = `https://app.circleci.com/pipelines/github/Clever/${repo}?branch=${branch}`;
  }

  const preText = failedStep
    ? `*:ohno: CI Failure*\n\n`
    : `*:successkid: CI Success*\n\n`;
  const repoText = `*Repo:*    \`${repo}\`\n`;
  const branchText = `*Branch:* \`${branch}\`\n`;
  const stateText = `*State:*    \`${state}\`\n`;
  const linkText = failedStep
    ? `*Link:*      <${targetURL}|Failed step>\n`
    : "";

  return preText + repoText + branchText + stateText + linkText;
}

function formAttachments(description, failedStep) {
  const text = failedStep
    ? `\n\`\`\`\n${description}\n\n${failedStep}\n\`\`\``
    : `\n\`\`\`\n${description}\n\`\`\``;
  const color = failedStep ? colorFailure : colorSuccess;
  const attachments = [
    {
      text,
      color,
    },
  ];

  return attachments;
}

runAction().then(() => null);

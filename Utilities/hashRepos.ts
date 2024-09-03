import simpleGit, { SimpleGit } from "simple-git";
import fs from "fs";
import path from "path";
import crypto from "crypto";

interface FileHash {
  file: string;
  hash: string;
}

// Function to clone the repositories and hash all files into a single hash
async function hashRepositories(
  repoUrls: string[],
  baseLocalPath: string
): Promise<string> {
  const git: SimpleGit = simpleGit();
  const hash = crypto.createHash("sha256");

  if (!fs.existsSync(baseLocalPath)) {
    fs.mkdirSync(baseLocalPath, { recursive: true });
  }

  for (const [index, repoUrl] of repoUrls.entries()) {
    const localPath = path.join(baseLocalPath, `repo-${index}`);
    await git.clone(repoUrl, localPath);

    const readFiles = (dir: string) => {
      const files = fs.readdirSync(dir);

      for (const file of files) {
        const filePath = path.join(dir, file);
        const stat = fs.statSync(filePath);

        if (stat.isDirectory()) {
          readFiles(filePath);
        } else {
          const content = fs.readFileSync(filePath);
          hash.update(content);
        }
      }
    };

    readFiles(localPath);

    // Clean up the cloned repository (optional)
    fs.rmSync(localPath, { recursive: true, force: true });
  }

  return hash.digest("hex");
}

// Main function to parse command-line arguments and call the hashRepositories function
async function main() {
  const args = process.argv.slice(2); // Remove the first two default args (node and script path)

  if (args.length < 2) {
    console.error(
      "Usage: node script.js <localPath> <repoUrl1> <repoUrl2> ... <repoUrlN>"
    );
    process.exit(1);
  }

  const baseLocalPath = args[0];
  const repoUrls = args.slice(1);

  try {
    const finalHash = await hashRepositories(repoUrls, baseLocalPath);
    console.log("Final Combined Hash:", finalHash);
  } catch (error) {
    console.error("Error during processing:", error);
  }
}

// Run the main function
main();

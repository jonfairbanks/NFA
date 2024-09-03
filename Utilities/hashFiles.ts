import fs from "fs";
import path from "path";
import crypto from "crypto";
import axios from "axios";

async function downloadAndHashFiles(
  uris: string[],
  localPath: string
): Promise<string> {
  const hash = crypto.createHash("sha256");

  if (!fs.existsSync(localPath)) {
    fs.mkdirSync(localPath, { recursive: true });
  }

  for (const [index, uri] of uris.entries()) {
    const fileName = path.join(localPath, `file-${index}`);
    const response = await axios.get(uri, { responseType: "arraybuffer" });

    // Save the file locally (optional)
    fs.writeFileSync(fileName, response.data);

    // Update the hash with the file content
    hash.update(response.data);

    // Optionally, clean up the downloaded file
    fs.unlinkSync(fileName);
  }

  return hash.digest("hex");
}

async function main() {
  const args = process.argv.slice(2);

  if (args.length < 2) {
    console.error(
      "Usage: ts-node hashFiles.ts <localPath> <uri1> <uri2> ... <uriN>"
    );
    process.exit(1);
  }

  const localPath = args[0];
  const uris = args.slice(1);

  try {
    const finalHash = await downloadAndHashFiles(uris, localPath);
    console.log("Final Combined Hash:", finalHash);
  } catch (error) {
    console.error("Error during processing:", error);
  }
}

main();

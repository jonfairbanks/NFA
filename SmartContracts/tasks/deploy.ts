import { deploy } from '../lib/deploy';

async function main() {
    await deploy();
  }
  
  main()
    .then(() => process.exit(0))
    .catch(error => {
      console.error(error);
      process.exit(1);
    });

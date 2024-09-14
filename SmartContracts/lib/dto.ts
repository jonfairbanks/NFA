class AppInfo {
     routerRequired: boolean;
     paymentModel: string;
     versionInfo: VersionInfo;

     constructor(routerRequired: boolean, paymentModel: string, downloadURIs: string[], codeHash: string, abiURIs: string[], abiHash: string[], versionId: string) {
         this.routerRequired = routerRequired;
         this.paymentModel = paymentModel;

         this.versionInfo = new VersionInfo(versionId, downloadURIs, codeHash, abiURIs, abiHash);
     }
}

class VersionInfo {
     versionId: string;
     downloadURIs: string[];
     codeHash: string;
     abiURIs: string[];
     abiHash: string;

     constructor(versionId: string, downloadURIs: string[], codeHash: string, abiURIs: string[], abiHash: string) {
         this.versionId = versionId;
         this.downloadURIs = downloadURIs;
         this.codeHash = codeHash;
         this.abiURIs = abiURIs;
         this.abiHash = abiHash;
     }
}
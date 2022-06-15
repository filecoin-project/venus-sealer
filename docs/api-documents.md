# Groups
* [](#)
  * [Closing](#Closing)
  * [Session](#Session)
  * [Shutdown](#Shutdown)
  * [Token](#Token)
  * [Version](#Version)
* [Actor](#Actor)
  * [ActorAddress](#ActorAddress)
  * [ActorAddressConfig](#ActorAddressConfig)
  * [ActorSectorSize](#ActorSectorSize)
* [Auth](#Auth)
  * [AuthNew](#AuthNew)
  * [AuthVerify](#AuthVerify)
* [Check](#Check)
  * [CheckProvable](#CheckProvable)
* [Compute](#Compute)
  * [ComputeDataCid](#ComputeDataCid)
  * [ComputeProof](#ComputeProof)
  * [ComputeWindowPoSt](#ComputeWindowPoSt)
* [Create](#Create)
  * [CreateBackup](#CreateBackup)
* [Current](#Current)
  * [CurrentSectorID](#CurrentSectorID)
* [Deal](#Deal)
  * [DealSector](#DealSector)
* [Deals](#Deals)
  * [DealsConsiderOfflineRetrievalDeals](#DealsConsiderOfflineRetrievalDeals)
  * [DealsConsiderOfflineStorageDeals](#DealsConsiderOfflineStorageDeals)
  * [DealsConsiderOnlineRetrievalDeals](#DealsConsiderOnlineRetrievalDeals)
  * [DealsConsiderOnlineStorageDeals](#DealsConsiderOnlineStorageDeals)
  * [DealsConsiderUnverifiedStorageDeals](#DealsConsiderUnverifiedStorageDeals)
  * [DealsConsiderVerifiedStorageDeals](#DealsConsiderVerifiedStorageDeals)
  * [DealsImportData](#DealsImportData)
  * [DealsList](#DealsList)
  * [DealsPieceCidBlocklist](#DealsPieceCidBlocklist)
  * [DealsSetConsiderOfflineRetrievalDeals](#DealsSetConsiderOfflineRetrievalDeals)
  * [DealsSetConsiderOfflineStorageDeals](#DealsSetConsiderOfflineStorageDeals)
  * [DealsSetConsiderOnlineRetrievalDeals](#DealsSetConsiderOnlineRetrievalDeals)
  * [DealsSetConsiderOnlineStorageDeals](#DealsSetConsiderOnlineStorageDeals)
  * [DealsSetConsiderUnverifiedStorageDeals](#DealsSetConsiderUnverifiedStorageDeals)
  * [DealsSetConsiderVerifiedStorageDeals](#DealsSetConsiderVerifiedStorageDeals)
  * [DealsSetPieceCidBlocklist](#DealsSetPieceCidBlocklist)
* [Get](#Get)
  * [GetDeals](#GetDeals)
* [Is](#Is)
  * [IsUnsealed](#IsUnsealed)
* [Log](#Log)
  * [LogList](#LogList)
  * [LogSetLevel](#LogSetLevel)
* [Mark](#Mark)
  * [MarkDealsAsPacking](#MarkDealsAsPacking)
* [Messager](#Messager)
  * [MessagerGetMessage](#MessagerGetMessage)
  * [MessagerPushMessage](#MessagerPushMessage)
  * [MessagerWaitMessage](#MessagerWaitMessage)
* [Mock](#Mock)
  * [MockWindowPoSt](#MockWindowPoSt)
* [Net](#Net)
  * [NetParamsConfig](#NetParamsConfig)
* [Pieces](#Pieces)
  * [PiecesGetCIDInfo](#PiecesGetCIDInfo)
  * [PiecesGetPieceInfo](#PiecesGetPieceInfo)
  * [PiecesListCidInfos](#PiecesListCidInfos)
  * [PiecesListPieces](#PiecesListPieces)
* [Pledge](#Pledge)
  * [PledgeSector](#PledgeSector)
* [Redo](#Redo)
  * [RedoSector](#RedoSector)
* [Return](#Return)
  * [ReturnAddPiece](#ReturnAddPiece)
  * [ReturnDataCid](#ReturnDataCid)
  * [ReturnFetch](#ReturnFetch)
  * [ReturnFinalizeReplicaUpdate](#ReturnFinalizeReplicaUpdate)
  * [ReturnFinalizeSector](#ReturnFinalizeSector)
  * [ReturnGenerateSectorKeyFromData](#ReturnGenerateSectorKeyFromData)
  * [ReturnMoveStorage](#ReturnMoveStorage)
  * [ReturnProveReplicaUpdate1](#ReturnProveReplicaUpdate1)
  * [ReturnProveReplicaUpdate2](#ReturnProveReplicaUpdate2)
  * [ReturnReadPiece](#ReturnReadPiece)
  * [ReturnReleaseUnsealed](#ReturnReleaseUnsealed)
  * [ReturnReplicaUpdate](#ReturnReplicaUpdate)
  * [ReturnSealCommit1](#ReturnSealCommit1)
  * [ReturnSealCommit2](#ReturnSealCommit2)
  * [ReturnSealPreCommit1](#ReturnSealPreCommit1)
  * [ReturnSealPreCommit2](#ReturnSealPreCommit2)
  * [ReturnUnsealPiece](#ReturnUnsealPiece)
* [Sealing](#Sealing)
  * [SealingAbort](#SealingAbort)
  * [SealingSchedDiag](#SealingSchedDiag)
* [Sector](#Sector)
  * [SectorAbortUpgrade](#SectorAbortUpgrade)
  * [SectorCommitFlush](#SectorCommitFlush)
  * [SectorCommitPending](#SectorCommitPending)
  * [SectorGetExpectedSealDuration](#SectorGetExpectedSealDuration)
  * [SectorGetSealDelay](#SectorGetSealDelay)
  * [SectorMarkForUpgrade](#SectorMarkForUpgrade)
  * [SectorMatchPendingPiecesToOpenSectors](#SectorMatchPendingPiecesToOpenSectors)
  * [SectorPreCommitFlush](#SectorPreCommitFlush)
  * [SectorPreCommitPending](#SectorPreCommitPending)
  * [SectorRemove](#SectorRemove)
  * [SectorSetExpectedSealDuration](#SectorSetExpectedSealDuration)
  * [SectorSetSealDelay](#SectorSetSealDelay)
  * [SectorStartSealing](#SectorStartSealing)
  * [SectorTerminate](#SectorTerminate)
  * [SectorTerminateFlush](#SectorTerminateFlush)
  * [SectorTerminatePending](#SectorTerminatePending)
* [Sectors](#Sectors)
  * [SectorsInfoListInStates](#SectorsInfoListInStates)
  * [SectorsList](#SectorsList)
  * [SectorsListInStates](#SectorsListInStates)
  * [SectorsRefs](#SectorsRefs)
  * [SectorsStatus](#SectorsStatus)
  * [SectorsSummary](#SectorsSummary)
  * [SectorsUnsealPiece](#SectorsUnsealPiece)
  * [SectorsUpdate](#SectorsUpdate)
* [Storage](#Storage)
  * [StorageAddLocal](#StorageAddLocal)
  * [StorageAttach](#StorageAttach)
  * [StorageBestAlloc](#StorageBestAlloc)
  * [StorageDeclareSector](#StorageDeclareSector)
  * [StorageDropSector](#StorageDropSector)
  * [StorageFindSector](#StorageFindSector)
  * [StorageGetLocks](#StorageGetLocks)
  * [StorageInfo](#StorageInfo)
  * [StorageList](#StorageList)
  * [StorageLocal](#StorageLocal)
  * [StorageLock](#StorageLock)
  * [StorageReportHealth](#StorageReportHealth)
  * [StorageStat](#StorageStat)
  * [StorageTryLock](#StorageTryLock)
* [Update](#Update)
  * [UpdateDealStatus](#UpdateDealStatus)
* [Worker](#Worker)
  * [WorkerConnect](#WorkerConnect)
  * [WorkerJobs](#WorkerJobs)
  * [WorkerStats](#WorkerStats)
## 


### Closing


Perms: read

Inputs: `null`

Response: `{}`

### Session


Perms: read

Inputs: `null`

Response: `"d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"`

### Shutdown


Perms: admin

Inputs: `null`

Response: `{}`

### Token


Perms: admin

Inputs: `null`

Response: `"Ynl0ZSBhcnJheQ=="`

### Version


Perms: read

Inputs: `null`

Response:
```json
{
  "Version": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
  "APIVersion": 66816,
  "BlockDelay": 42
}
```

## Actor


### ActorAddress
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response: `"t01234"`

### ActorAddressConfig
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response:
```json
{
  "PreCommitControl": [
    "t01234"
  ],
  "CommitControl": [
    "t01234"
  ],
  "TerminateControl": [
    "t01234"
  ],
  "DealPublishControl": [
    "t01234"
  ],
  "DisableOwnerFallback": true,
  "DisableWorkerFallback": true
}
```

### ActorSectorSize
There are not yet any comments for this method.

Perms: read

Inputs:
```json
[
  "t01234"
]
```

Response: `34359738368`

## Auth


### AuthNew


Perms: admin

Inputs:
```json
[
  [
    "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  ]
]
```

Response: `"Ynl0ZSBhcnJheQ=="`

### AuthVerify


Perms: read

Inputs:
```json
[
  "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
]
```

Response:
```json
[
  "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
]
```

## Check


### CheckProvable
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  8,
  [
    {
      "ID": {
        "Miner": 1000,
        "Number": 9
      },
      "ProofType": 8
    }
  ],
  true
]
```

Response:
```json
{
  "123": "can't acquire read lock"
}
```

## Compute


### ComputeDataCid
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  1024,
  {}
]
```

Response:
```json
{
  "Size": 1032,
  "PieceCID": {
    "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
  }
}
```

### ComputeProof
There are not yet any comments for this method.

Perms: read

Inputs:
```json
[
  [
    {
      "SealProof": 8,
      "SectorNumber": 9,
      "SectorKey": null,
      "SealedCID": {
        "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
      }
    }
  ],
  "Bw==",
  10101,
  14
]
```

Response:
```json
[
  {
    "PoStProof": 8,
    "ProofBytes": "Ynl0ZSBhcnJheQ=="
  }
]
```

### ComputeWindowPoSt
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  42,
  []
]
```

Response:
```json
[
  {
    "Deadline": 42,
    "Partitions": [
      {
        "Index": 42,
        "Skipped": [
          5,
          1
        ]
      }
    ],
    "Proofs": [
      {
        "PoStProof": 8,
        "ProofBytes": "Ynl0ZSBhcnJheQ=="
      }
    ],
    "ChainCommitEpoch": 10101,
    "ChainCommitRand": "Bw=="
  }
]
```

## Create


### CreateBackup
CreateBackup creates node backup onder the specified file name. The
method requires that the venus-sealer is running with the
LOTUS_BACKUP_BASE_PATH environment variable set to some path, and that
the path specified when calling CreateBackup is within the base path


Perms: admin

Inputs:
```json
[
  "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
]
```

Response: `{}`

## Current


### CurrentSectorID
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response: `9`

## Deal


### DealSector
There are not yet any comments for this method.

Perms: admin

Inputs: `null`

Response:
```json
[
  {
    "DealId": 5432,
    "SectorId": 9,
    "PieceCid": {
      "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
    },
    "Offset": 1032,
    "Size": 1032
  }
]
```

## Deals


### DealsConsiderOfflineRetrievalDeals
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response: `true`

### DealsConsiderOfflineStorageDeals
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response: `true`

### DealsConsiderOnlineRetrievalDeals
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response: `true`

### DealsConsiderOnlineStorageDeals
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response: `true`

### DealsConsiderUnverifiedStorageDeals
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response: `true`

### DealsConsiderVerifiedStorageDeals
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response: `true`

### DealsImportData
There are not yet any comments for this method.

Perms: write

Inputs:
```json
[
  {
    "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
  },
  "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
]
```

Response: `{}`

### DealsList
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response:
```json
[
  {
    "Proposal": {
      "PieceCID": {
        "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
      },
      "PieceSize": 1032,
      "VerifiedDeal": true,
      "Client": "t01234",
      "Provider": "t01234",
      "Label": "",
      "StartEpoch": 10101,
      "EndEpoch": 10101,
      "StoragePricePerEpoch": "0",
      "ProviderCollateral": "0",
      "ClientCollateral": "0"
    },
    "State": {
      "SectorStartEpoch": 10101,
      "LastUpdatedEpoch": 10101,
      "SlashEpoch": 10101
    }
  }
]
```

### DealsPieceCidBlocklist
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response:
```json
[
  {
    "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
  }
]
```

### DealsSetConsiderOfflineRetrievalDeals
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  true
]
```

Response: `{}`

### DealsSetConsiderOfflineStorageDeals
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  true
]
```

Response: `{}`

### DealsSetConsiderOnlineRetrievalDeals
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  true
]
```

Response: `{}`

### DealsSetConsiderOnlineStorageDeals
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  true
]
```

Response: `{}`

### DealsSetConsiderUnverifiedStorageDeals
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  true
]
```

Response: `{}`

### DealsSetConsiderVerifiedStorageDeals
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  true
]
```

Response: `{}`

### DealsSetPieceCidBlocklist
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  [
    {
      "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
    }
  ]
]
```

Response: `{}`

## Get


### GetDeals
for market


Perms: admin

Inputs:
```json
[
  123,
  123
]
```

Response:
```json
[
  {
    "DealID": 5432,
    "SectorID": 9,
    "Offset": 1032,
    "Length": 1032,
    "Proposal": {
      "PieceCID": {
        "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
      },
      "PieceSize": 1032,
      "VerifiedDeal": true,
      "Client": "t01234",
      "Provider": "t01234",
      "Label": "",
      "StartEpoch": 10101,
      "EndEpoch": 10101,
      "StoragePricePerEpoch": "0",
      "ProviderCollateral": "0",
      "ClientCollateral": "0"
    },
    "ClientSignature": {
      "Type": 2,
      "Data": "Ynl0ZSBhcnJheQ=="
    },
    "TransferType": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
    "Root": {
      "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
    },
    "PublishCid": {
      "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
    },
    "FastRetrieval": true,
    "Status": ""
  }
]
```

## Is


### IsUnsealed
There are not yet any comments for this method.

Perms: read

Inputs:
```json
[
  {
    "ID": {
      "Miner": 1000,
      "Number": 9
    },
    "ProofType": 8
  },
  1023477,
  1024
]
```

Response: `true`

## Log


### LogList


Perms: write

Inputs: `null`

Response:
```json
[
  "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
]
```

### LogSetLevel


Perms: write

Inputs:
```json
[
  "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
  "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
]
```

Response: `{}`

## Mark


### MarkDealsAsPacking
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  [
    5432
  ]
]
```

Response: `{}`

## Messager


### MessagerGetMessage
There are not yet any comments for this method.

Perms: write

Inputs:
```json
[
  "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
]
```

Response:
```json
{
  "ID": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
  "UnsignedCid": null,
  "SignedCid": null,
  "Version": 42,
  "To": "t01234",
  "From": "t01234",
  "Nonce": 42,
  "Value": "0",
  "GasLimit": 9,
  "GasFeeCap": "0",
  "GasPremium": "0",
  "Method": 1,
  "Params": "Ynl0ZSBhcnJheQ==",
  "Signature": {
    "Type": 2,
    "Data": "Ynl0ZSBhcnJheQ=="
  },
  "Height": 9,
  "Confidence": 9,
  "Receipt": {
    "ExitCode": 0,
    "Return": "Ynl0ZSBhcnJheQ==",
    "GasUsed": 9
  },
  "TipSetKey": [],
  "Meta": {
    "expireEpoch": 10101,
    "gasOverEstimation": 12.3,
    "maxFee": "0",
    "maxFeeCap": "0"
  },
  "WalletName": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
  "FromUser": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
  "State": 0,
  "CreatedAt": "0001-01-01T00:00:00Z",
  "UpdatedAt": "0001-01-01T00:00:00Z"
}
```

### MessagerPushMessage
There are not yet any comments for this method.

Perms: sign

Inputs:
```json
[
  {
    "CID": {
      "/": "bafy2bzacebbpdegvr3i4cosewthysg5xkxpqfn2wfcz6mv2hmoktwbdxkax4s"
    },
    "Version": 42,
    "To": "t01234",
    "From": "t01234",
    "Nonce": 42,
    "Value": "0",
    "GasLimit": 9,
    "GasFeeCap": "0",
    "GasPremium": "0",
    "Method": 1,
    "Params": "Ynl0ZSBhcnJheQ=="
  },
  {
    "expireEpoch": 10101,
    "gasOverEstimation": 12.3,
    "maxFee": "0",
    "maxFeeCap": "0"
  }
]
```

Response: `"63f292b3-b804-4e59-86d4-f4c2fd3e275a"`

### MessagerWaitMessage
messager


Perms: read

Inputs:
```json
[
  "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
  42
]
```

Response:
```json
{
  "Message": {
    "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
  },
  "Receipt": {
    "ExitCode": 0,
    "Return": "Ynl0ZSBhcnJheQ==",
    "GasUsed": 9
  },
  "ReturnDec": {},
  "TipSet": [],
  "Height": 10101
}
```

## Mock


### MockWindowPoSt
Test WdPoSt


Perms: write

Inputs:
```json
[
  [
    {
      "SealProof": 8,
      "SectorNumber": 9,
      "SectorKey": null,
      "SealedCID": {
        "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
      }
    }
  ],
  "Bw=="
]
```

Response: `{}`

## Net


### NetParamsConfig
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response:
```json
{
  "UpgradeIgnitionHeight": 10101,
  "UpgradeOhSnapHeight": 9,
  "ForkLengthThreshold": 10101,
  "BlockDelaySecs": 42,
  "PreCommitChallengeDelay": 10101
}
```

## Pieces


### PiecesGetCIDInfo
There are not yet any comments for this method.

Perms: read

Inputs:
```json
[
  {
    "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
  }
]
```

Response:
```json
{
  "CID": {
    "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
  },
  "PieceBlockLocations": [
    {
      "RelOffset": 42,
      "BlockSize": 42,
      "PieceCID": {
        "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
      }
    }
  ]
}
```

### PiecesGetPieceInfo
There are not yet any comments for this method.

Perms: read

Inputs:
```json
[
  {
    "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
  }
]
```

Response:
```json
{
  "PieceCID": {
    "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
  },
  "Deals": [
    {
      "DealID": 5432,
      "SectorID": 9,
      "Offset": 1032,
      "Length": 1032
    }
  ]
}
```

### PiecesListCidInfos
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response:
```json
[
  {
    "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
  }
]
```

### PiecesListPieces
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response:
```json
[
  {
    "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
  }
]
```

## Pledge


### PledgeSector
Temp api for testing


Perms: write

Inputs: `null`

Response:
```json
{
  "Miner": 1000,
  "Number": 9
}
```

## Redo


### RedoSector
Redo


Perms: write

Inputs:
```json
[
  {
    "SectorNumber": 9,
    "SealPath": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
    "StorePath": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

## Return


### ReturnAddPiece


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  {
    "Size": 1032,
    "PieceCID": {
      "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
    }
  },
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnDataCid


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  {
    "Size": 1032,
    "PieceCID": {
      "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
    }
  },
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnFetch


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnFinalizeReplicaUpdate


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnFinalizeSector


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnGenerateSectorKeyFromData


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnMoveStorage


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnProveReplicaUpdate1


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  [
    "Ynl0ZSBhcnJheQ=="
  ],
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnProveReplicaUpdate2


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  "Bw==",
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnReadPiece


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  true,
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnReleaseUnsealed


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnReplicaUpdate


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  {
    "NewSealed": {
      "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
    },
    "NewUnsealed": {
      "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
    }
  },
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnSealCommit1


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  "Bw==",
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnSealCommit2


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  "Bw==",
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnSealPreCommit1


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  "Bw==",
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnSealPreCommit2


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  {
    "Unsealed": {
      "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
    },
    "Sealed": {
      "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
    }
  },
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### ReturnUnsealPiece


Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  },
  {
    "Code": 0,
    "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

## Sealing


### SealingAbort
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  {
    "Sector": {
      "Miner": 1000,
      "Number": 9
    },
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
  }
]
```

Response: `{}`

### SealingSchedDiag
SealingSchedDiag dumps internal sealing scheduler state


Perms: admin

Inputs:
```json
[
  true
]
```

Response: `{}`

## Sector


### SectorAbortUpgrade
SectorAbortUpgrade can be called on sectors that are in the process of being upgraded to abort it


Perms: admin

Inputs:
```json
[
  9
]
```

Response: `{}`

### SectorCommitFlush
SectorCommitFlush immediately sends a Commit message with sectors aggregated for Commit.
Returns null if message wasn't sent


Perms: admin

Inputs: `null`

Response:
```json
[
  {
    "Sectors": [
      123,
      124
    ],
    "FailedSectors": {
      "123": "can't acquire read lock"
    },
    "Msg": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
    "Error": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

### SectorCommitPending
SectorCommitPending returns a list of pending Commit sectors to be sent in the next aggregate message


Perms: admin

Inputs: `null`

Response:
```json
[
  {
    "Miner": 1000,
    "Number": 9
  }
]
```

### SectorGetExpectedSealDuration
SectorGetExpectedSealDuration gets the expected time for a sector to seal


Perms: read

Inputs: `null`

Response: `60000000000`

### SectorGetSealDelay
SectorGetSealDelay gets the time that a newly-created sector
waits for more deals before it starts sealing


Perms: read

Inputs: `null`

Response: `60000000000`

### SectorMarkForUpgrade
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  9,
  true
]
```

Response: `{}`

### SectorMatchPendingPiecesToOpenSectors


Perms: admin

Inputs: `null`

Response: `{}`

### SectorPreCommitFlush
SectorPreCommitFlush immediately sends a PreCommit message with sectors batched for PreCommit.
Returns null if message wasn't sent


Perms: admin

Inputs: `null`

Response:
```json
[
  {
    "Sectors": [
      123,
      124
    ],
    "Msg": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
    "Error": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

### SectorPreCommitPending
SectorPreCommitPending returns a list of pending PreCommit sectors to be sent in the next batch message


Perms: admin

Inputs: `null`

Response:
```json
[
  {
    "Miner": 1000,
    "Number": 9
  }
]
```

### SectorRemove
SectorRemove removes the sector from storage. It doesn't terminate it on-chain, which can
be done with SectorTerminate. Removing and not terminating live sectors will cause additional penalties.


Perms: admin

Inputs:
```json
[
  9
]
```

Response: `{}`

### SectorSetExpectedSealDuration
SectorSetExpectedSealDuration sets the expected time for a sector to seal


Perms: write

Inputs:
```json
[
  60000000000
]
```

Response: `{}`

### SectorSetSealDelay
SectorSetSealDelay sets the time that a newly-created sector
waits for more deals before it starts sealing


Perms: write

Inputs:
```json
[
  60000000000
]
```

Response: `{}`

### SectorStartSealing
SectorStartSealing can be called on sectors in Empty or WaitDeals states
to trigger sealing early


Perms: write

Inputs:
```json
[
  9
]
```

Response: `{}`

### SectorTerminate
SectorTerminate terminates the sector on-chain (adding it to a termination batch first), then
automatically removes it from storage


Perms: admin

Inputs:
```json
[
  9
]
```

Response: `{}`

### SectorTerminateFlush
SectorTerminateFlush immediately sends a terminate message with sectors batched for termination.
Returns null if message wasn't sent


Perms: admin

Inputs: `null`

Response: `"63f292b3-b804-4e59-86d4-f4c2fd3e275a"`

### SectorTerminatePending
SectorTerminatePending returns a list of pending sector terminations to be sent in the next batch message


Perms: admin

Inputs: `null`

Response:
```json
[
  {
    "Miner": 1000,
    "Number": 9
  }
]
```

## Sectors


### SectorsInfoListInStates
List all staged sector's info in particular states


Perms: read

Inputs:
```json
[
  [
    "PreCommit1"
  ],
  true,
  true
]
```

Response:
```json
[
  {
    "SectorID": 9,
    "State": "PreCommit1",
    "CommD": null,
    "CommR": null,
    "Proof": "Ynl0ZSBhcnJheQ==",
    "Deals": [
      5432
    ],
    "Pieces": [
      {
        "Piece": {
          "Size": 1032,
          "PieceCID": {
            "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
          }
        },
        "DealInfo": {
          "PublishCid": null,
          "DealID": 5432,
          "DealProposal": {
            "PieceCID": {
              "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
            },
            "PieceSize": 1032,
            "VerifiedDeal": true,
            "Client": "t01234",
            "Provider": "t01234",
            "Label": "",
            "StartEpoch": 10101,
            "EndEpoch": 10101,
            "StoragePricePerEpoch": "0",
            "ProviderCollateral": "0",
            "ClientCollateral": "0"
          },
          "DealSchedule": {
            "StartEpoch": 10101,
            "EndEpoch": 10101
          },
          "KeepUnsealed": true
        }
      }
    ],
    "Ticket": {
      "Value": "Bw==",
      "Epoch": 10101
    },
    "Seed": {
      "Value": "Bw==",
      "Epoch": 10101
    },
    "PreCommitMsg": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
    "CommitMsg": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
    "Retries": 42,
    "ToUpgrade": true,
    "CCUpdate": true,
    "UpdateSealed": null,
    "UpdateUnsealed": null,
    "ReplicaUpdateMessage": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
    "LastErr": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
    "Log": [
      {
        "Kind": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
        "Timestamp": 42,
        "Trace": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
        "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
      }
    ],
    "SealProof": 8,
    "Activation": 10101,
    "Expiration": 10101,
    "DealWeight": "0",
    "VerifiedDealWeight": "0",
    "InitialPledge": "0",
    "OnTime": 10101,
    "Early": 10101
  }
]
```

### SectorsList
List all staged sectors


Perms: read

Inputs: `null`

Response:
```json
[
  123,
  124
]
```

### SectorsListInStates
List sectors in particular states


Perms: read

Inputs:
```json
[
  [
    "PreCommit1"
  ]
]
```

Response:
```json
[
  123,
  124
]
```

### SectorsRefs
There are not yet any comments for this method.

Perms: read

Inputs: `null`

Response:
```json
{
  "10": [
    {
      "SectorID": 9,
      "Offset": 1032,
      "Size": 1024
    }
  ]
}
```

### SectorsStatus
Get the status of a given sector by ID


Perms: read

Inputs:
```json
[
  9,
  true
]
```

Response:
```json
{
  "SectorID": 9,
  "State": "PreCommit1",
  "CommD": null,
  "CommR": null,
  "Proof": "Ynl0ZSBhcnJheQ==",
  "Deals": [
    5432
  ],
  "Pieces": [
    {
      "Piece": {
        "Size": 1032,
        "PieceCID": {
          "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
        }
      },
      "DealInfo": {
        "PublishCid": null,
        "DealID": 5432,
        "DealProposal": {
          "PieceCID": {
            "/": "bafy2bzacea3wsdh6y3a36tb3skempjoxqpuyompjbmfeyf34fi3uy6uue42v4"
          },
          "PieceSize": 1032,
          "VerifiedDeal": true,
          "Client": "t01234",
          "Provider": "t01234",
          "Label": "",
          "StartEpoch": 10101,
          "EndEpoch": 10101,
          "StoragePricePerEpoch": "0",
          "ProviderCollateral": "0",
          "ClientCollateral": "0"
        },
        "DealSchedule": {
          "StartEpoch": 10101,
          "EndEpoch": 10101
        },
        "KeepUnsealed": true
      }
    }
  ],
  "Ticket": {
    "Value": "Bw==",
    "Epoch": 10101
  },
  "Seed": {
    "Value": "Bw==",
    "Epoch": 10101
  },
  "PreCommitMsg": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
  "CommitMsg": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
  "Retries": 42,
  "ToUpgrade": true,
  "CCUpdate": true,
  "UpdateSealed": null,
  "UpdateUnsealed": null,
  "ReplicaUpdateMessage": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
  "LastErr": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
  "Log": [
    {
      "Kind": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
      "Timestamp": 42,
      "Trace": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
      "Message": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
    }
  ],
  "SealProof": 8,
  "Activation": 10101,
  "Expiration": 10101,
  "DealWeight": "0",
  "VerifiedDealWeight": "0",
  "InitialPledge": "0",
  "OnTime": 10101,
  "Early": 10101
}
```

### SectorsSummary
Get summary info of sectors


Perms: read

Inputs: `null`

Response:
```json
{
  "PreCommit1": 0
}
```

### SectorsUnsealPiece
SectorsUnsealPiece will Unseal a Sealed sector file for the given sector.


Perms: write

Inputs:
```json
[
  {
    "ID": {
      "Miner": 1000,
      "Number": 9
    },
    "ProofType": 8
  },
  1023477,
  1024,
  "Bw==",
  null
]
```

Response: `{}`

### SectorsUpdate
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  9,
  "PreCommit1"
]
```

Response: `{}`

## Storage


### StorageAddLocal
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
]
```

Response: `{}`

### StorageAttach


Perms: admin

Inputs:
```json
[
  {
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab",
    "URLs": [
      "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
    ],
    "Weight": 42,
    "MaxStorage": 42,
    "CanSeal": true,
    "CanStore": true,
    "Groups": [
      "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
    ],
    "AllowTo": [
      "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
    ]
  },
  {
    "Capacity": 9,
    "Available": 9,
    "FSAvailable": 9,
    "Reserved": 9,
    "Max": 9,
    "Used": 9
  }
]
```

Response: `{}`

### StorageBestAlloc


Perms: admin

Inputs:
```json
[
  1,
  34359738368,
  "sealing"
]
```

Response:
```json
[
  {
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab",
    "URLs": [
      "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
    ],
    "Weight": 42,
    "MaxStorage": 42,
    "CanSeal": true,
    "CanStore": true,
    "Groups": [
      "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
    ],
    "AllowTo": [
      "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
    ]
  }
]
```

### StorageDeclareSector


Perms: admin

Inputs:
```json
[
  "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab",
  {
    "Miner": 1000,
    "Number": 9
  },
  1,
  true
]
```

Response: `{}`

### StorageDropSector


Perms: admin

Inputs:
```json
[
  "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab",
  {
    "Miner": 1000,
    "Number": 9
  },
  1
]
```

Response: `{}`

### StorageFindSector


Perms: admin

Inputs:
```json
[
  {
    "Miner": 1000,
    "Number": 9
  },
  1,
  34359738368,
  true
]
```

Response:
```json
[
  {
    "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab",
    "URLs": [
      "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
    ],
    "Weight": 42,
    "CanSeal": true,
    "CanStore": true,
    "Primary": true
  }
]
```

### StorageGetLocks


Perms: admin

Inputs: `null`

Response:
```json
{
  "Locks": [
    {
      "Sector": {
        "Miner": 1000,
        "Number": 9
      },
      "Write": [
        3,
        3,
        3,
        3,
        3
      ],
      "Read": [
        3,
        3,
        3,
        3,
        3
      ]
    }
  ]
}
```

### StorageInfo


Perms: admin

Inputs:
```json
[
  "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
]
```

Response:
```json
{
  "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab",
  "URLs": [
    "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  ],
  "Weight": 42,
  "MaxStorage": 42,
  "CanSeal": true,
  "CanStore": true,
  "Groups": [
    "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  ],
  "AllowTo": [
    "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  ]
}
```

### StorageList


Perms: admin

Inputs: `null`

Response:
```json
{
  "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab": [
    {
      "Miner": 1000,
      "Number": 9,
      "SectorFileType": 1
    }
  ]
}
```

### StorageLocal
There are not yet any comments for this method.

Perms: admin

Inputs: `null`

Response:
```json
{
  "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab": "local path"
}
```

### StorageLock


Perms: admin

Inputs:
```json
[
  {
    "Miner": 1000,
    "Number": 9
  },
  1,
  1
]
```

Response: `{}`

### StorageReportHealth


Perms: admin

Inputs:
```json
[
  "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab",
  {
    "Stat": {
      "Capacity": 9,
      "Available": 9,
      "FSAvailable": 9,
      "Reserved": 9,
      "Max": 9,
      "Used": 9
    },
    "Err": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
  }
]
```

Response: `{}`

### StorageStat
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
]
```

Response:
```json
{
  "Capacity": 9,
  "Available": 9,
  "FSAvailable": 9,
  "Reserved": 9,
  "Max": 9,
  "Used": 9
}
```

### StorageTryLock


Perms: admin

Inputs:
```json
[
  {
    "Miner": 1000,
    "Number": 9
  },
  1,
  1
]
```

Response: `true`

## Update


### UpdateDealStatus
There are not yet any comments for this method.

Perms: admin

Inputs:
```json
[
  5432,
  "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
]
```

Response: `{}`

## Worker


### WorkerConnect
WorkerConnect tells the node to connect to workers RPC


Perms: admin

Inputs:
```json
[
  "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
]
```

Response: `{}`

### WorkerJobs
There are not yet any comments for this method.

Perms: admin

Inputs: `null`

Response:
```json
{
  "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab": [
    {
      "ID": {
        "Sector": {
          "Miner": 1000,
          "Number": 9
        },
        "ID": "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab"
      },
      "Sector": {
        "Miner": 1000,
        "Number": 9
      },
      "Task": "seal/v0/addpiece",
      "RunWait": 123,
      "Start": "0001-01-01T00:00:00Z",
      "Hostname": "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
    }
  ]
}
```

### WorkerStats
There are not yet any comments for this method.

Perms: admin

Inputs: `null`

Response:
```json
{
  "d5c7e3cb-f35a-4f98-b509-ca8ce5922fab": {
    "Info": {
      "Hostname": "63f292b3-b804-4e59-86d4-f4c2fd3e275a",
      "IgnoreResources": true,
      "Resources": {
        "MemPhysical": 42,
        "MemUsed": 42,
        "MemSwap": 42,
        "MemSwapUsed": 42,
        "CPUs": 42,
        "GPUs": [
          "63f292b3-b804-4e59-86d4-f4c2fd3e275a"
        ],
        "Resources": {
          "seal/v0/addpiece": {
            "8": {
              "MinMemory": 42,
              "MaxMemory": 42,
              "GPUUtilization": 12.3,
              "MaxParallelism": 123,
              "MaxParallelismGPU": 123,
              "BaseMinMemory": 42
            }
          }
        }
      }
    },
    "Enabled": true,
    "MemUsedMin": 42,
    "MemUsedMax": 42,
    "GpuUsed": 12.3,
    "CpuUse": 42
  }
}
```


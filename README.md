# tariff-lambdas-electronic-tariff-file-rotations

Scheduled go lambda function to rotate reports in the reporting bucket
(including the Electronic Tariff files).

```mermaid
sequenceDiagram
  participant Scheduler as Scheduler
  participant Lambda as Lambda
  participant AWS S3 Bucket as AWS S3 Bucket

  Scheduler->>Lambda: Trigger at 08:00 AM
  Lambda->>AWS S3 Bucket: Perform ListObjectV2 on bucket files
  Lambda->>Lambda: Extract candidates for deletion
  Lambda->>AWS S3 Bucket: Perform DeleteObject on candidate files
```

## License

[MIT License](LICENSE)

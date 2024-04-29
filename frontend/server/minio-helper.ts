import { Stream } from 'stream';
// Copyright 2019-2020 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import { Transform, PassThrough } from 'stream';
import * as tar from 'tar-stream';
import peek from 'peek-stream';
import gunzip from 'gunzip-maybe';
import { URL } from 'url';
import { Client as MinioClient, ClientOptions as MinioClientOptions } from 'minio';
import { awsInstanceProfileCredentials, isAWSS3Endpoint } from './aws-helper';
import { S3ProviderInfo} from "./handlers/artifacts";
import {getK8sSecret} from "./k8s-helper";
import {parseJSONString} from "./utils";
const { fromNodeProviderChain } = require('@aws-sdk/credential-providers');

/** MinioRequestConfig describes the info required to retrieve an artifact. */
export interface MinioRequestConfig {
  bucket: string;
  key: string;
  client: MinioClient;
  tryExtract?: boolean;
}

/** MinioClientOptionsWithOptionalSecrets wraps around MinioClientOptions where only endPoint is required (accesskey and secretkey are optional). */
export interface MinioClientOptionsWithOptionalSecrets extends Partial<MinioClientOptions> {
  endPoint: string;
}

/**
 * Create minio client for s3 compatible storage
 *
 * If providerInfoString is available, use these over defaultConfigs.
 *
 * If providerInfo is not provided or, if credentials are sourced fromEnv,
 * then, if using aws s3 (via provider chain or instance profile), create a
 * minio client backed by aws s3 client.
 *
 * Otherwise, assume s3 compatible credentials have been provided via configs
 * (defaultConfigs or ProviderInfo), and return a minio client configured
 * respectively.
 *
 * @param defaultConfig minio client options where `accessKey` and `secretKey` are optional.
 * @param providerType provider type ('s3' or 'minio')
 * @param providerInfoString json string container optional provider info
 */
export async function createMinioClient(defaultConfig: MinioClientOptionsWithOptionalSecrets, providerType: string, providerInfoString?: string) {
  let config = defaultConfig;

  if (providerInfoString) {
    const providerInfo  = parseJSONString<S3ProviderInfo>(providerInfoString);
    // If fromEnv == false, we rely on the default credentials or env to provide credentials (e.g. IRSA)
    if (providerInfo && providerInfo.Params.fromEnv === "false") {
      config = await parseS3ProviderInfo(config, providerInfo);
    }
  }
  // If using s3 and sourcing credentials from environment (currently only check aws env)
  if (providerType === "s3" && (!config.accessKey || !config.secretKey)) {
    // AWS S3 with credentials from provider chain
    if (isAWSS3Endpoint(config.endPoint)) {
      try {
        const credentials = fromNodeProviderChain();
        const aws_credentials = await credentials();
        if (aws_credentials) {
          const {
            accessKeyId: accessKey,
            secretAccessKey: secretKey,
            sessionToken: sessionToken,
          } = aws_credentials;
          return new MinioClient({ ...config, accessKey, secretKey, sessionToken });
        }
      } catch (err) {
        console.error('Unable to get credentials from AWS credential provider chain: ', err);
      }
    } else {
      // If no access/secret key or endpoint provided
      // attempt to fetch AWS S3 instance profile credentials
      if (!config.accessKey || !config.secretKey) {
        try {
          if (await awsInstanceProfileCredentials.ok()) {
            const credentials = await awsInstanceProfileCredentials.getCredentials();
            if (credentials) {
              const {
                AccessKeyId: accessKey,
                SecretAccessKey: secretKey,
                Token: sessionToken,
              } = credentials;
              return new MinioClient({ ...config, accessKey, secretKey, sessionToken });
            }
            console.error('Unable to get credentials from AWS metadata store.');
          }
        } catch (err) {
          console.error('Unable to get aws instance profile credentials: ', err);
        }
      }
    }
  }

  // If using any AWS or S3 compatible store (e.g. minio, aws s3 when using manual creds, ceph, etc.)
  let mc : MinioClient;
  try {
    mc = await new MinioClient(config as MinioClientOptions);
  } catch (err) {
    throw new Error(`Failed to create MinioClient: ${err}`);
  }
  return mc;
}

// Parse provider info for any s3 compatible store that's not AWS S3
async function parseS3ProviderInfo(config: MinioClientOptionsWithOptionalSecrets, providerInfo: S3ProviderInfo) : Promise<MinioClientOptionsWithOptionalSecrets> {
  if (!providerInfo.Params.accessKeyKey || !providerInfo.Params.secretKeyKey || !providerInfo.Params.secretName) {
    throw new Error('Provider info with fromEnv:false supplied with incomplete secret credential info.');
  }
  config.accessKey = await getK8sSecret(providerInfo.Params.secretName, providerInfo.Params.accessKeyKey);
  config.secretKey = await getK8sSecret(providerInfo.Params.secretName, providerInfo.Params.secretKeyKey);

  if (isAWSS3Endpoint(providerInfo.Params.endpoint)) {
    if (providerInfo.Params.endpoint) {
      if(providerInfo.Params.endpoint.startsWith("https")){
        const parseEndpoint = new URL(providerInfo.Params.endpoint);
        config.endPoint = parseEndpoint.hostname;
      } else {
        config.endPoint = providerInfo.Params.endpoint;
      }

    } else {
      throw new Error('Provider info missing endpoint parameter.');
    }

    if (providerInfo.Params.region) {
      config.region = providerInfo.Params.region;
    }

    // It's possible the user specifies these via config
    // since aws s3 and s3-compatible use the same config parameters
    // safeguard the user by ensuring these remain unset (default)
    config.port = undefined;
    config.useSSL = undefined;
  } else {
    if (providerInfo.Params.endpoint) {
      const parseEndpoint = new URL(providerInfo.Params.endpoint);
      const host = parseEndpoint.hostname;
      const port = parseEndpoint.port;
      config.endPoint = host;
      // user provided port in endpoint takes precedence
      // e.g. if the user has provided <service-name>.<namespace>.svc.cluster.local:<service-port>
      config.port = port ? Number(port) : undefined;
    }

    config.region = providerInfo.Params.region ? providerInfo.Params.region : undefined;

    if (providerInfo.Params.disableSSL) {
      config.useSSL = !(providerInfo.Params.disableSSL.toLowerCase() === "true");
    } else {
      config.useSSL = undefined;
    }
  }
  return config;
}

/**
 * Checks the magic number of a buffer to see if the mime type is a uncompressed
 * tarball. The buffer must be of length 264 bytes or more.
 *
 * See also: https://www.gnu.org/software/tar/manual/html_node/Standard.html
 *
 * @param buf Buffer
 */
export function isTarball(buf: Buffer) {
  if (!buf || buf.length < 264) {
    return false;
  }
  const offset = 257;
  const v1 = [0x75, 0x73, 0x74, 0x61, 0x72, 0x00, 0x30, 0x30];
  const v0 = [0x75, 0x73, 0x74, 0x61, 0x72, 0x20, 0x20, 0x00];

  return (
    v1.reduce((res, curr, i) => res && curr === buf[offset + i], true) ||
    v0.reduce((res, curr, i) => res && curr === buf[offset + i], true as boolean)
  );
}

/**
 * Returns a stream that extracts the first record of a tarball if the source
 * stream is a tarball, otherwise just pipe the content as is.
 */
export function maybeTarball(): Transform {
  return peek(
    { newline: false, maxBuffer: 264 },
    (data: Buffer, swap: (error?: Error, parser?: Transform) => void) => {
      if (isTarball(data)) swap(undefined, extractFirstTarRecordAsStream());
      else swap(undefined, new PassThrough());
    },
  );
}

/**
 * Returns a transform stream where the first record inside a tarball will be
 * pushed - i.e. all other contents will be dropped.
 */
function extractFirstTarRecordAsStream() {
  const extract = tar.extract();
  const transformStream = new Transform({
    write: (chunk: any, _encoding: string, callback: (error?: Error | null) => void) => {
      extract.write(chunk, callback);
    },
  });
  extract.once('entry', function(_header, stream, next) {
    stream.on('data', (buffer: any) => transformStream.push(buffer));
    stream.on('end', () => {
      transformStream.emit('end');
      next();
    });
    stream.resume(); // just auto drain the stream
  });
  extract.on('error', error => transformStream.emit('error', error));
  return transformStream;
}

/**
 * Returns a stream from an object in a s3 compatible object store (e.g. minio).
 * The actual content of the stream depends on the object.
 *
 * Any gzipped or deflated objects will be ungzipped or inflated. If the object
 * is a tarball, only the content of the first record in the tarball will be
 * returned. For any other objects, the raw content will be returned.
 *
 * @param param.bucket Bucket name to retrieve the object from.
 * @param param.key Key of the object to retrieve.
 * @param param.client Minio client.
 * @param param.tryExtract Whether we try to extract *.tar.gz, default to true.
 *
 */
export async function getObjectStream({
  bucket,
  key,
  client,
  tryExtract = true,
}: MinioRequestConfig): Promise<Transform> {
  const stream = await client.getObject(bucket, key);
  return tryExtract ? stream.pipe(gunzip()).pipe(maybeTarball()) : stream.pipe(new PassThrough());
}

import Centrifuge from 'centrifuge/dist/centrifuge';
import { LiveDataStreamOptions, toDataQueryError } from '@grafana/runtime';
import { BehaviorSubject, Observable } from 'rxjs';
import {
  DataFrame,
  DataFrameJSON,
  dataFrameToJSON,
  DataQueryResponse,
  isLiveChannelMessageEvent,
  isLiveChannelStatusEvent,
  LiveChannelAddress,
  LiveChannelConfig,
  LiveChannelConnectionState,
  LiveChannelEvent,
  LiveChannelPresenceStatus,
  LoadingState,
  StreamingDataFrame,
} from '@grafana/data';
import { CentrifugeLiveChannel } from './channel';
import { liveTimer } from 'app/features/dashboard/dashgrid/liveTimer';

type CentrifugeSrvDeps = {
  appUrl: string;
  orgId: number;
  orgRole: string;
  sessionId: string;
  liveEnabled: boolean;
};

export class CentrifugeSrv {
  readonly open = new Map<string, CentrifugeLiveChannel>();
  readonly centrifuge: Centrifuge;
  readonly connectionState: BehaviorSubject<boolean>;
  readonly connectionBlocker: Promise<void>;

  constructor(private deps: CentrifugeSrvDeps) {
    const liveUrl = `${deps.appUrl.replace(/^http/, 'ws')}/api/live/ws`;
    this.centrifuge = new Centrifuge(liveUrl, {});
    this.centrifuge.setConnectData({
      sessionId: deps.sessionId,
      orgId: deps.orgId,
    });
    // orgRole is set when logged in *or* anonomus users can use grafana
    if (deps.liveEnabled && deps.orgRole !== '') {
      this.centrifuge.connect(); // do connection
    }
    this.connectionState = new BehaviorSubject<boolean>(this.centrifuge.isConnected());
    this.connectionBlocker = new Promise<void>((resolve) => {
      if (this.centrifuge.isConnected()) {
        return resolve();
      }
      const connectListener = () => {
        resolve();
        this.centrifuge.removeListener('connect', connectListener);
      };
      this.centrifuge.addListener('connect', connectListener);
    });

    // Register global listeners
    this.centrifuge.on('connect', this.onConnect);
    this.centrifuge.on('disconnect', this.onDisconnect);
    this.centrifuge.on('publish', this.onServerSideMessage);
  }

  //----------------------------------------------------------
  // Internal functions
  //----------------------------------------------------------

  onConnect = (context: any) => {
    this.connectionState.next(true);
  };

  onDisconnect = (context: any) => {
    this.connectionState.next(false);
  };

  onServerSideMessage = (context: any) => {
    console.log('Publication from server-side channel', context);
  };

  /**
   * Get a channel.  If the scope, namespace, or path is invalid, a shutdown
   * channel will be returned with an error state indicated in its status
   */
  getChannel<TMessage>(addr: LiveChannelAddress, config: LiveChannelConfig): CentrifugeLiveChannel<TMessage> {
    const id = `${this.deps.orgId}/${addr.scope}/${addr.namespace}/${addr.path}`;
    let channel = this.open.get(id);
    if (channel != null) {
      return channel;
    }

    channel = new CentrifugeLiveChannel(id, addr);
    channel.shutdownCallback = () => {
      this.open.delete(id); // remove it from the list of open channels
    };
    this.open.set(id, channel);

    // Initialize the channel in the background
    this.initChannel(config, channel).catch((err) => {
      if (channel) {
        channel.currentStatus.state = LiveChannelConnectionState.Invalid;
        channel.shutdownWithError(err);
      }
      this.open.delete(id);
    });

    // return the not-yet initalized channel
    return channel;
  }

  private async initChannel(config: LiveChannelConfig, channel: CentrifugeLiveChannel): Promise<void> {
    const events = channel.initalize(config);
    if (!this.centrifuge.isConnected()) {
      await this.connectionBlocker;
    }
    channel.subscription = this.centrifuge.subscribe(channel.id, events);
    return;
  }

  //----------------------------------------------------------
  // Exported functions
  //----------------------------------------------------------

  /**
   * Listen for changes to the connection state
   */
  getConnectionState() {
    return this.connectionState.asObservable();
  }

  /**
   * Watch for messages in a channel
   */
  getStream<T>(address: LiveChannelAddress, config: LiveChannelConfig): Observable<LiveChannelEvent<T>> {
    return this.getChannel<T>(address, config).getStream();
  }

  /**
   * Connect to a channel and return results as DataFrames
   */
  getDataStream(options: LiveDataStreamOptions, config: LiveChannelConfig): Observable<DataQueryResponse> {
    return new Observable<DataQueryResponse>((subscriber) => {
      const channel = this.getChannel(options.addr, config);
      const key = options.key ?? `xstr/${streamCounter++}`;
      let data: StreamingDataFrame | undefined = undefined;
      let filtered: DataFrame | undefined = undefined;
      let state = LoadingState.Streaming;
      let last = liveTimer.lastUpdate;
      let lastWidth = -1;

      const process = (msg: DataFrameJSON) => {
        if (!data) {
          data = new StreamingDataFrame(msg, options.buffer);
        } else {
          data.push(msg);
        }
        state = LoadingState.Streaming;
        const sameWidth = lastWidth === data.fields.length;
        lastWidth = data.fields.length;

        // Filter out fields
        if (!filtered || msg.schema || !sameWidth) {
          filtered = data;
          if (options.filter) {
            const { fields } = options.filter;
            if (fields?.length) {
              filtered = {
                ...data,
                fields: data.fields.filter((f) => fields.includes(f.name)),
              };
            }
          }
        }

        const elapsed = liveTimer.lastUpdate - last;
        if (elapsed > 1000 || liveTimer.ok) {
          filtered.length = data.length; // make sure they stay up-to-date
          subscriber.next({ state, data: [filtered], key });
          last = liveTimer.lastUpdate;
        }
      };

      if (options.frame) {
        process(dataFrameToJSON(options.frame));
      } else if (channel.lastMessageWithSchema) {
        process(channel.lastMessageWithSchema);
      }

      const sub = channel.getStream().subscribe({
        error: (err: any) => {
          console.log('LiveQuery [error]', { err }, options.addr);
          state = LoadingState.Error;
          subscriber.next({ state, data: [data], key, error: toDataQueryError(err) });
          sub.unsubscribe(); // close after error
        },
        complete: () => {
          console.log('LiveQuery [complete]', options.addr);
          if (state !== LoadingState.Error) {
            state = LoadingState.Done;
          }
          // or track errors? subscriber.next({ state, data: [data], key });
          subscriber.complete();
          sub.unsubscribe();
        },
        next: (evt: LiveChannelEvent) => {
          if (isLiveChannelMessageEvent(evt)) {
            process(evt.message);
            return;
          }
          if (isLiveChannelStatusEvent(evt)) {
            if (evt.error) {
              let error = toDataQueryError(evt.error);
              error.message = `Streaming channel error: ${error.message}`;
              state = LoadingState.Error;
              subscriber.next({ state, data: [data], key, error });
              return;
            } else if (
              evt.state === LiveChannelConnectionState.Connected ||
              evt.state === LiveChannelConnectionState.Pending
            ) {
              if (evt.message) {
                process(evt.message);
              }
              return;
            }
            console.log('ignore state', evt);
          }
        },
      });

      return () => {
        sub.unsubscribe();
      };
    });
  }

  /**
   * For channels that support presence, this will request the current state from the server.
   *
   * Join and leave messages will be sent to the open stream
   */
  getPresence(address: LiveChannelAddress, config: LiveChannelConfig): Promise<LiveChannelPresenceStatus> {
    return this.getChannel(address, config).getPresence();
  }
}

// This is used to give a unique key for each stream.  The actual value does not matter
let streamCounter = 0;

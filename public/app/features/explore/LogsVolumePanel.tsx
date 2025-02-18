import { AbsoluteTimeRange, DataQueryResponse, GrafanaTheme2, LoadingState, SplitOpen, TimeZone } from '@grafana/data';
import { Alert, Button, Collapse, TooltipDisplayMode, useStyles2, useTheme2 } from '@grafana/ui';
import { ExploreGraph } from './ExploreGraph';
import React from 'react';
import { css } from '@emotion/css';

type Props = {
  logsVolumeData?: DataQueryResponse;
  absoluteRange: AbsoluteTimeRange;
  timeZone: TimeZone;
  splitOpen: SplitOpen;
  width: number;
  onUpdateTimeRange: (timeRange: AbsoluteTimeRange) => void;
  onLoadLogsVolume: () => void;
};

export function LogsVolumePanel(props: Props) {
  const { width, logsVolumeData, absoluteRange, timeZone, splitOpen, onUpdateTimeRange, onLoadLogsVolume } = props;
  const theme = useTheme2();
  const styles = useStyles2(getStyles);
  const spacing = parseInt(theme.spacing(2).slice(0, -2), 10);
  const height = 150;

  let LogsVolumePanelContent;

  if (!logsVolumeData) {
    return null;
  } else if (logsVolumeData?.error) {
    return (
      <Alert title="Failed to load volume logs for this query">
        {logsVolumeData.error.data?.message || logsVolumeData.error.statusText || logsVolumeData.error.message}
      </Alert>
    );
  } else if (logsVolumeData?.state === LoadingState.Loading) {
    LogsVolumePanelContent = <span>Logs volume is loading...</span>;
  } else if (logsVolumeData?.data) {
    if (logsVolumeData.data.length > 0) {
      LogsVolumePanelContent = (
        <ExploreGraph
          graphStyle="lines"
          loadingState={LoadingState.Done}
          data={logsVolumeData.data}
          height={height}
          width={width - spacing}
          absoluteRange={absoluteRange}
          onChangeTime={onUpdateTimeRange}
          timeZone={timeZone}
          splitOpenFn={splitOpen}
          tooltipDisplayMode={TooltipDisplayMode.Multi}
        />
      );
    } else {
      LogsVolumePanelContent = <span>No volume data.</span>;
    }
  }

  const zoomRatio = logsLevelZoomRatio(logsVolumeData, absoluteRange);
  let zoomLevelInfo;

  if (zoomRatio !== undefined && zoomRatio < 1) {
    zoomLevelInfo = (
      <>
        <span className={styles.zoomInfo}>Reload logs volume</span>
        <Button size="xs" icon="sync" variant="secondary" onClick={onLoadLogsVolume} />
      </>
    );
  }

  return (
    <Collapse label="Logs volume" isOpen={true} loading={logsVolumeData?.state === LoadingState.Loading}>
      <div style={{ height }} className={styles.contentContainer}>
        {LogsVolumePanelContent}
      </div>
      <div className={styles.zoomInfoContainer}>{zoomLevelInfo}</div>
    </Collapse>
  );
}

const getStyles = (theme: GrafanaTheme2) => {
  return {
    zoomInfoContainer: css`
      display: flex;
      justify-content: end;
      position: absolute;
      right: 5px;
      top: 5px;
    `,
    zoomInfo: css`
      padding: 8px;
      font-size: ${theme.typography.bodySmall.fontSize};
    `,
    contentContainer: css`
      display: flex;
      align-items: center;
      justify-content: center;
    `,
  };
};

function logsLevelZoomRatio(
  logsVolumeData: DataQueryResponse | undefined,
  selectedTimeRange: AbsoluteTimeRange
): number | undefined {
  const dataRange = logsVolumeData && logsVolumeData.data[0] && logsVolumeData.data[0].meta?.custom?.absoluteRange;
  return dataRange ? (selectedTimeRange.from - selectedTimeRange.to) / (dataRange.from - dataRange.to) : undefined;
}

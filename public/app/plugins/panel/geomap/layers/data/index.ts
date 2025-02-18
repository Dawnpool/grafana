import { markersLayer } from './markersLayer';
import { geojsonMapper } from './geojsonMapper';
import { heatmapLayer } from './heatMap';
import { lastPointTracker } from './lastPointTracker';
import { textLabelsLayer } from './textLabelsLayer';

/**
 * Registry for layer handlers
 */
export const dataLayers = [
  markersLayer,
  heatmapLayer,
  lastPointTracker,
  geojsonMapper, // dummy for now
  textLabelsLayer,
];

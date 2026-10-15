import type { ApiResponse } from '../../scripts/api';
import { apiGet, apiPost } from '../../scripts/api';
import type { RisqTerrainType } from '../game/games/risq/rendering/terrain';
import type { GameRisqFromServer } from '../game/games/risq/transport/snapshot_types';
import type { MapDoc } from './map_doc';

export interface TerrainEntry {
  terrain_id: number;
  display_name: string;
  terrain_type: RisqTerrainType;
}

export function listTerrains(config_dir?: string): Promise<ApiResponse<TerrainEntry[]>> {
  return apiGet<TerrainEntry[]>(config_dir ? `dev/risq/terrains/${config_dir}` : 'dev/risq/terrains');
}

export interface EntityEntry {
  id: number;
  display_name: string;
}

export interface EntityCatalog {
  buildings: EntityEntry[];
  units: EntityEntry[];
  resources: EntityEntry[];
}

export function listEntities(config_dir?: string): Promise<ApiResponse<EntityCatalog>> {
  return apiGet<EntityCatalog>(config_dir ? `dev/risq/entities/${config_dir}` : 'dev/risq/entities');
}

export function listMaps(): Promise<ApiResponse<string[]>> {
  return apiGet<string[]>('dev/risq/maps');
}

export function loadMap(name: string): Promise<ApiResponse<MapDoc>> {
  return apiGet<MapDoc>(`dev/risq/maps/${name}`);
}

export function saveMap(name: string, doc: MapDoc): Promise<ApiResponse<boolean>> {
  return apiPost<boolean>(`dev/risq/maps/${name}`, doc);
}

export function deleteMap(name: string): Promise<ApiResponse<boolean>> {
  return apiPost<boolean>(`dev/risq/maps/${name}/delete`, {});
}

export function previewMap(doc: MapDoc): Promise<ApiResponse<GameRisqFromServer>> {
  return apiPost<GameRisqFromServer>('dev/risq/preview', doc);
}

import { z } from "zod"

export const equipmentQualitySchema = z.number().int().min(0).max(25)

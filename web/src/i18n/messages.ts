import { commonEn, commonZhCN } from './catalogs/common'
import { managementEn, managementZhCN } from './catalogs/management'
import { elementsEn, elementsZhCN } from './catalogs/elements'
import { operationsEn, operationsZhCN } from './catalogs/operations'
import { docsEn, docsZhCN } from './catalogs/docs'

export const en = { ...commonEn, ...managementEn, ...elementsEn, ...operationsEn, ...docsEn }
export type MessageKey = keyof typeof en
export const zhCN: Record<MessageKey, string> = { ...commonZhCN, ...managementZhCN, ...elementsZhCN, ...operationsZhCN, ...docsZhCN }

/**
 * 获取数据库表列表（分页）
 *
 * @param {Object} dbConfig - 数据库配置
 * @param {number} [page=1] - 页码（从1开始）
 * @param {number} [pageSize=30] - 每页数量
 * @param {string} [keyword] - 表名模糊搜索关键字
 * @returns {Object} { tables: string[], total: number, page: number, pageSize: number }
 */
function getTables(dbConfig, page, pageSize, keyword) {
    if (typeof page !== 'number' || typeof pageSize !== 'number') {
        return { error: 'page and pageSize must be number' };
    }
    page = page || 1;
    pageSize = pageSize || 30;

    var offset = (page - 1) * pageSize;
    var totalQuery, tablesQuery;

    switch (dbConfig.driver) {
        case 'sqlite3':
            var keywordClause = keyword ? " AND name LIKE '%" + keyword + "%'" : "";
            totalQuery = "SELECT COUNT(*) as total FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'" + keywordClause;
            tablesQuery = "SELECT name as tableName FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'" + keywordClause + " ORDER BY name LIMIT " + pageSize + " OFFSET " + offset;
            break;
        case 'mysql':
            var keywordClause = keyword ? " AND table_name LIKE '%" + keyword + "%'" : "";
            totalQuery = "SELECT COUNT(*) as total FROM information_schema.tables WHERE table_schema = DATABASE()" + keywordClause;
            tablesQuery = "SELECT table_name as tableName FROM information_schema.tables WHERE table_schema = DATABASE()" + keywordClause + " ORDER BY table_name LIMIT " + pageSize + " OFFSET " + offset;
            break;
        default:
            return { error: 'unsupported driver: ' + dbConfig.driver };
    }

    var db = sql.open({
        driverName: dbConfig.driver,
        dsn: dbConfig.dsn
    });

    var total = db.query(totalQuery).rows[0].total;
    var tables = db.query(tablesQuery).rows.map(function(row) {
        return String(row.tableName);
    });
    db.close();

    return {
        tables: tables,
        total: total,
        page: page,
        pageSize: pageSize
    };
}

module.exports = {
    type: 'tool',
    name: 'list-tables',
    description: '分页查询指定数据库的表列表',
    inputSchema: {
        type: 'object',
        properties: {
            name: {
                type: 'string',
                description: '数据库名称（精确匹配，用于定位数据库配置）'
            },
            page: {
                type: 'number',
                description: '页码（从1开始）',
                default: 1
            },
            pageSize: {
                type: 'number',
                description: '每页数量',
                default: 30
            },
            keyword: {
                type: 'string',
                description: '表名模糊搜索关键字'
            }
        },
        required: ['name']
    },
    execute: function (params) {
        var name = params.name;
        var page = params.page || 1;
        var pageSize = params.pageSize || 30;
        var keyword = params.keyword;

        var dbConfig = getDb(name);
        if (!dbConfig) {
            return { error: 'database not found: ' + name };
        }

        return getTables(dbConfig, page, pageSize, keyword);
    }
};

function getDb(name) {
    var dbList = config.get("database").toSlice();
    for (var i = 0; i < dbList.length; i++) {
        if (dbList[i].name === name) {
            return dbList[i];
        }
    }
    return null;
}
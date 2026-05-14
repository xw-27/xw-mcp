/**
 * 获取数据库列表（分页）
 *
 * @param {number} [page=1] - 页码（从1开始）
 * @param {number} [pageSize=30] - 每页数量
 * @param {string} [name] - 数据库名称（模糊搜索）
 * @param {string} [driver] - 数据库类型（模糊搜索）
 * @returns {Object} { databases: Object[], total: number, page: number, pageSize: number }
 */
function listDatabases(page, pageSize, name, driver) {
    if (typeof page !== 'number' || typeof pageSize !== 'number') {
        return { error: 'page and pageSize must be number' };
    }
    page = page || 1;
    pageSize = pageSize || 30;

    var allDatabases = config.get("database").toSlice();

    if (name) {
        allDatabases = allDatabases.filter(function(db) {
            return db.name.indexOf(name) !== -1;
        });
    }

    if (driver) {
        allDatabases = allDatabases.filter(function(db) {
            return db.driver.indexOf(driver) !== -1;
        });
    }

    var total = allDatabases.length;
    var offset = (page - 1) * pageSize;
    var pageDatabases = allDatabases.slice(offset, offset + pageSize);

    pageDatabases = pageDatabases.map(function(db) {
        return { name: db.name, driver: db.driver };
    });

    return {
        databases: pageDatabases,
        total: total,
        page: page,
        pageSize: pageSize
    };
}

module.exports = {
    type: 'tool',
    name: 'list-databases',
    description: '分页查询配置的数据库列表',
    inputSchema: {
        type: 'object',
        properties: {
            name: {
                type: 'string',
                description: '数据库名称模糊搜索关键字'
            },
            driver: {
                type: 'string',
                description: '数据库类型模糊搜索关键字（如 sqlite3、mysql）'
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
            }
        }
    },
    execute: function (params) {
        var name = params.name;
        var driver = params.driver;
        var page = params.page || 1;
        var pageSize = params.pageSize || 30;

        return listDatabases(page, pageSize, name, driver);
    }
};
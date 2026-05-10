module.exports = {
    type: 'tool',
    name: 'hello',
    description: '测试工具',
    inputSchema: {
        type: 'object',
        properties: {
            name: { type: 'string', description: '名字' }
        },
        required: ['name']
    },
    execute: function(params) {
        return { message: 'Hello' + params.name }
    }
}
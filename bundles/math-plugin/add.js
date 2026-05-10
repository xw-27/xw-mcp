module.exports = {
    type: 'tool',
    name: 'add',
    description: '两个数字相加',
    inputSchema: {
        type: 'object',
        properties: {
            augend: {
                type: 'number',
                description: '被加数'
            },
            addend: {
                type: 'number',
                description: '加数'
            }
        },
        required: ['augend', 'addend']
    },
    execute: function(params) {
        return { result: params.augend + params.addend };
    }
}
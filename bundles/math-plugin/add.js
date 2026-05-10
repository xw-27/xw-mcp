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
        const result = params.augend + params.addend;
        log.info(`add called: ${params.augend} + ${params.addend} = ${result}`);
        return { result: result };
    }
}
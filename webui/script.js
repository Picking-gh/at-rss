document.addEventListener('DOMContentLoaded', () => {
    const taskListUl = document.getElementById('task-list');
    const taskDetailPanel = document.getElementById('task-detail-panel');
    const taskFormContainer = document.getElementById('task-form-container');
    const addTaskBtn = document.getElementById('add-task-btn');
    const modal = document.getElementById('modal');
    const modalTitle = document.getElementById('modal-title');
    const modalBody = document.getElementById('modal-body');
    const closeModalBtn = document.querySelector('.close-button');

    let currentTasks = {}; // Store fetched tasks {name: config}
    let selectedTaskName = null; // Keep track of the currently selected task

    // --- API Helper Functions ---

    async function apiFetch(url, options = {}) {
        const response = await fetch(url, options);
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`HTTP error! status: ${response.status}, message: ${errorText}`);
        }
        const contentType = response.headers.get("content-type");
        return contentType?.includes("application/json")
            ? await response.json()
            : await response.text();
    }

    // --- Task List Management ---

    async function loadTasks() {
        currentTasks = await apiFetch('/api/tasks').catch(() => {
            taskListUl.innerHTML = '<li>Failed to load tasks.</li>';
            return {};
        });
        renderTaskList();
        clearTaskDetailPanel();
    }

    function renderTaskList() {
        taskListUl.innerHTML = ''; // Clear existing list
        const sortedTaskNames = Object.keys(currentTasks).sort();

        if (sortedTaskNames.length === 0) {
            taskListUl.innerHTML = '<li>No tasks configured yet.</li>';
            return;
        }

        sortedTaskNames.forEach(taskName => {
            const li = document.createElement('li');
            const button = document.createElement('button');
            button.textContent = taskName;
            button.classList.add('task-button');
            button.dataset.taskName = taskName;
            if (taskName === selectedTaskName) {
                button.classList.add('active');
            }
            if (currentTasks[taskName].isNew) {
                button.classList.add('new-task');
            }
            button.addEventListener('click', () => selectTask(taskName));
            li.appendChild(button);
            taskListUl.appendChild(li);
        });
    }

    function selectTask(taskName) {
        selectedTaskName = taskName;
        renderTaskList(); // Re-render list to highlight the active task
        renderTaskDetail(taskName);
    }

    function clearTaskDetailPanel() {
        taskFormContainer.innerHTML = '<p>Please select a task from the list or add a new one.</p>';
        selectedTaskName = null;
        renderTaskList();
    }

    // --- Helper to get current task context ---
    function getCurrentTaskContext() {
        const form = document.getElementById('task-form');
        if (!form) return null;

        const taskName = form.dataset.taskName;
        const isNew = form.dataset.isNew === 'true';

        if (!taskName && !isNew) {
            alert('Internal Error: Task context is missing. Please select a task or start adding a new one.');
            return null;
        }

        const taskConfig = currentTasks[taskName];
        if (!taskConfig && !isNew) {
            alert(`Internal Error: Configuration for task "${taskName}" not found.`);
            return null;
        }

        return { taskName, isNew, taskConfig };
    }


    // --- Task Detail Rendering ---

    function renderTaskDetail(taskName) {
        const taskConfig = currentTasks[taskName];
        if (!taskConfig) {
            clearTaskDetailPanel();
            alert(`Task "${taskName}" not found in currentTasks.`);
            return;
        }

        taskFormContainer.innerHTML = '';

        const form = document.createElement('form');
        form.id = 'task-form';
        form.dataset.taskName = taskName;
        form.dataset.isNew = taskConfig.isNew;

        // --- Basic Info ---
        const infoSection = createFormSection(form, 'Basic Info');
        createTextField(infoSection, 'taskNameDisplay', 'Task Name', taskName, true);
        createNumberField(infoSection, 'interval', 'Fetch Interval (minutes)', taskConfig.interval || 10);

        // --- Downloaders ---
        renderDownloaderSection(form, taskConfig.downloaders || []);

        // --- Feed URLs ---
        renderFeedSection(form, taskConfig.feed?.URLs || []);

        // --- Filter ---
        renderFilterSection(form, taskConfig.filter);

        // --- Extracter ---
        renderExtracterSection(form, taskConfig.extracter);


        // --- Action Buttons ---
        const actionDiv = document.createElement('div');
        actionDiv.classList.add('action-buttons');

        const saveButton = document.createElement('button');
        saveButton.type = 'submit';
        saveButton.textContent = form.dataset.isNew === "true" ? 'Create Task' : 'Save Changes';
        saveButton.classList.add('button', 'primary-button');
        actionDiv.appendChild(saveButton);

        const deleteButton = document.createElement('button');
        deleteButton.type = 'button'; // Important: prevent form submission
        deleteButton.textContent = 'Delete Task';
        deleteButton.classList.add('button', 'danger-button');
        deleteButton.addEventListener('click', () => deleteTask(taskName));
        actionDiv.appendChild(deleteButton);

        form.appendChild(actionDiv);
        taskFormContainer.appendChild(form);

        form.addEventListener('submit', handleFormSubmit);
    }

    // --- Form Element Creation Helpers ---

    function createFormSection(parent, title) {
        const section = document.createElement('div');
        section.classList.add('form-section');
        const h3 = document.createElement('h3');
        h3.textContent = title;
        section.appendChild(h3);
        parent.appendChild(section);
        return section;
    }

    function createInputField(parent, id, label, options = {}) {
        const { type = 'text', value = '', readOnly = false, min, placeholder } = options;
        const group = document.createElement('div');
        group.classList.add('form-group');

        const labelEl = document.createElement('label');
        labelEl.htmlFor = id;
        labelEl.textContent = label;

        const input = document.createElement('input');
        input.type = type;
        input.id = id;
        input.name = id;
        input.value = value;
        input.readOnly = readOnly;

        if (min !== undefined) input.min = min;
        if (placeholder !== undefined) input.placeholder = placeholder;
        if (readOnly) input.style.backgroundColor = '#eee';

        group.appendChild(labelEl);
        group.appendChild(input);
        parent.appendChild(group);
        return input;
    }

    function createTextField(parent, id, label, value = '', readOnly = false, placeholder) {
        return createInputField(parent, id, label, { type: 'text', value, readOnly, placeholder });
    }

    function createPasswordField(parent, id, label, value = '') {
        return createInputField(parent, id, label, { type: 'password', value });
    }

    function createNumberField(parent, id, label, value = '', readOnly = false, placeholder) {
        return createInputField(parent, id, label, { type: 'number', value, readOnly, min: 1, placeholder });
    }

    function createCheckboxField(parent, id, label, checked = false) {
        const group = document.createElement('div');
        group.classList.add('form-group', 'checkbox-group');

        const input = document.createElement('input');
        input.type = 'checkbox';
        input.id = id;
        input.name = id;
        input.checked = checked;

        const labelEl = document.createElement('label');
        labelEl.htmlFor = id;
        labelEl.textContent = label;

        group.append(input, labelEl);
        parent.appendChild(group);
        return input;
    }

    function createSelectField(parent, id, label, options, selectedValue = '') {
        const group = document.createElement('div');
        group.classList.add('form-group');

        const labelEl = document.createElement('label');
        labelEl.htmlFor = id;
        labelEl.textContent = label;

        const select = document.createElement('select');
        select.id = id;
        select.name = id;

        options.forEach(opt => {
            const optionEl = document.createElement('option');
            optionEl.value = opt.value;
            optionEl.textContent = opt.text;
            optionEl.selected = opt.value === selectedValue;
            select.appendChild(optionEl);
        });

        group.append(labelEl, select);
        parent.appendChild(group);
        return select;
    }

    function createListSection(parent, title, items, renderItemFunc, addItemFunc, listId) {
        const section = createFormSection(parent, title);
        const ul = document.createElement('ul');
        ul.id = listId;
        ul.classList.add('list-items');
        items.forEach((item, index) => renderItemFunc(ul, item, index));
        section.appendChild(ul);

        const addButton = document.createElement('button');
        addButton.type = 'button';
        addButton.textContent = `Add ${title.replace(' List', '')}`;
        addButton.classList.add('button', 'secondary-button', 'add-item-button');
        // addButton.style.marginTop = '10px';
        addButton.addEventListener('click', () => addItemFunc(ul));
        section.appendChild(addButton);
        return ul;
    }


    // --- Section Rendering Functions ---

    function renderDownloaderSection(form, downloaders) {
        const listId = 'downloader-list';
        createListSection(form, 'Downloaders', downloaders, renderDownloaderItem, addDownloader, listId);
    }

    function renderDownloaderItem(ul, downloader, index) {
        const li = document.createElement('li');
        li.dataset.index = index;
        li.innerHTML = `
            <span><strong>Type:</strong> ${downloader.type} | <strong>Host:</strong> ${downloader.host || 'default'}:${downloader.port || 'default'}</span>
            <div class="list-item-actions">
                <button type="button" class="edit-downloader-btn button secondary-button">Edit</button>
                <button type="button" class="delete-downloader-btn button danger-button">Del</button>
            </div>
        `;
        li.querySelector('.edit-downloader-btn').addEventListener('click', () => editDownloader(index));
        li.querySelector('.delete-downloader-btn').addEventListener('click', () => deleteDownloader(index));
        ul.appendChild(li);
    }

    function addDownloader(ul) {
        // Add Downloader handler
        const context = getCurrentTaskContext();
        if (!context) return;
        const { taskName, isNew, taskConfig } = context;
        const targetConfig = taskConfig || (isNew ? currentTasks[taskName] : null);

        if (targetConfig) {
            openDownloaderModal();
        } else {
            console.error(`addDownloader: Could not find target config for task "${taskName}" (isNew: ${isNew})`);
        }
    }

    function editDownloader(index) {
        // Edit Downloader handler
        const context = getCurrentTaskContext();
        if (!context) return;
        const { taskName } = context;
        const downloaderData = currentTasks[taskName]?.downloaders[index];
        if (downloaderData) {
            openDownloaderModal(downloaderData, index);
        } else {
            alert("Could not find downloader data to edit.");
        }
    }

    function deleteDownloader(index) {
        // Delete Downloader handler
        deleteTaskListItem('downloaders', index, `Are you sure you want to delete downloader #${index + 1}?`);
    }


    function renderFeedSection(form, urls) {
        const section = createFormSection(form, 'Feed URLs');
        const listId = 'feed-url-list';
        const ul = document.createElement('ul');
        ul.id = listId;
        ul.classList.add('list-items');
        urls.forEach((url, index) => renderFeedUrlItem(ul, url, index));
        section.appendChild(ul);

        const addForm = document.createElement('div');
        addForm.classList.add('add-item-form');
        const urlInput = document.createElement('input');
        urlInput.type = 'url';
        urlInput.placeholder = 'Enter new feed URL';
        urlInput.id = 'new-feed-url';
        const addButton = document.createElement('button');
        addButton.type = 'button';
        addButton.textContent = 'Add URL';
        addButton.classList.add('button', 'secondary-button');
        addButton.addEventListener('click', () => {
            const newUrl = urlInput.value.trim();
            if (newUrl) {
                addFeedUrl(ul, newUrl);
                urlInput.value = '';
            } else {
                alert('Please enter a valid URL.');
            }
        });
        addForm.appendChild(urlInput);
        addForm.appendChild(addButton);
        section.appendChild(addForm);
    }

    function renderFeedUrlItem(ul, url, index) {
        const li = document.createElement('li');
        li.dataset.index = index;
        li.innerHTML = `
            <span>${url}</span>
            <div class="list-item-actions">
                 <button type="button" class="delete-feed-url-btn button danger-button">Del</button>
            </div>
        `;
        li.querySelector('.delete-feed-url-btn').addEventListener('click', () => deleteFeedUrl(index));
        ul.appendChild(li);
    }

    function addFeedUrl(ul, newUrl) {
        // Use the generic add function
        // Ensure the path exists before adding
        const context = getCurrentTaskContext();
        if (!context) return;
        const { taskName, taskConfig, isNew } = context;
        const targetConfig = taskConfig || (isNew ? currentTasks[taskName] : null);

        if (targetConfig) {
            if (!targetConfig.feed) {
                targetConfig.feed = { URLs: [] };
            }
            if (!targetConfig.feed.URLs) {
                targetConfig.feed.URLs = [];
            }
            addTaskListItem('feed.URLs', newUrl);
        } else {
            console.error(`addFeedUrl: Could not find target config for task "${taskName}" (isNew: ${isNew})`);
        }
    }


    function deleteFeedUrl(index) {
        // Use the generic delete function
        deleteTaskListItem('feed.URLs', index, 'Are you sure you want to delete this feed URL?');
    }


    function renderFilterSection(form, filter) {
        const section = createFormSection(form, 'Filter');
        if (!filter) {
            const addButton = document.createElement('button');
            addButton.type = 'button';
            addButton.textContent = 'Add Filter';
            addButton.classList.add('button', 'secondary-button');
            addButton.addEventListener('click', () => {
                // Use generic toggle function
                toggleTaskSection('filter', { include: [], exclude: [] });
            });
            section.appendChild(addButton);
            return;
        }

        renderKeywordList(section, 'Include Keywords', filter.include || [], 'include-keywords', addIncludeKeyword, deleteIncludeKeyword);
        renderKeywordList(section, 'Exclude Keywords', filter.exclude || [], 'exclude-keywords', addExcludeKeyword, deleteExcludeKeyword);

        const removeFilterBtn = document.createElement('button');
        removeFilterBtn.type = 'button';
        removeFilterBtn.textContent = 'Remove Filter Section';
        removeFilterBtn.classList.add('button', 'danger-button');
        // removeFilterBtn.style.marginTop = '10px';
        removeFilterBtn.addEventListener('click', () => {
            // Use generic toggle function
            toggleTaskSection('filter', null, 'Are you sure you want to remove the entire filter section?');
        });
        section.appendChild(removeFilterBtn);
    }

    function renderKeywordList(parent, title, keywords, listId, addFunc, deleteFunc) {
        const subSection = document.createElement('div');
        subSection.classList.add('form-subsection');
        const h4 = document.createElement('h4');
        h4.textContent = title;
        subSection.appendChild(h4);

        const ul = document.createElement('ul');
        ul.id = listId;
        ul.classList.add('list-items');
        keywords.forEach((keyword, index) => renderKeywordItem(ul, keyword, index, deleteFunc));
        subSection.appendChild(ul);

        const addForm = document.createElement('div');
        addForm.classList.add('add-item-form');
        const keywordInput = document.createElement('input');
        keywordInput.type = 'text';
        keywordInput.placeholder = 'Enter new keyword(s)';
        const addButton = document.createElement('button');
        addButton.type = 'button';
        addButton.textContent = 'Add';
        addButton.classList.add('button', 'secondary-button');
        addButton.addEventListener('click', () => {
            const newKeyword = keywordInput.value.trim();
            if (newKeyword) {
                addFunc(ul, newKeyword);
                keywordInput.value = '';
            } else {
                alert('Please enter a keyword.');
            }
        });
        addForm.appendChild(keywordInput);
        addForm.appendChild(addButton);
        subSection.appendChild(addForm);

        parent.appendChild(subSection);
    }

    function renderKeywordItem(ul, keyword, index, deleteFunc) {
        const li = document.createElement('li');
        li.dataset.index = index;
        li.innerHTML = `
            <span>${keyword}</span>
            <div class="list-item-actions">
                 <button type="button" class="delete-keyword-btn button danger-button">Del</button>
            </div>
        `;
        li.querySelector('.delete-keyword-btn').addEventListener('click', () => deleteFunc(index));
        ul.appendChild(li);
    }

    function addIncludeKeyword(ul, keyword) {
        // Ensure filter and include array exist before adding
        const context = getCurrentTaskContext();
        if (!context) return;
        const { taskName, taskConfig, isNew } = context;
        const targetConfig = taskConfig || (isNew ? currentTasks[taskName] : null);

        if (targetConfig?.filter) {
            if (!targetConfig.filter.include) targetConfig.filter.include = [];
            addTaskListItem('filter.include', keyword);
        } else {
            console.warn(`addIncludeKeyword: Filter section does not exist for task "${taskName}". Cannot add keyword.`);
        }
    }
    function deleteIncludeKeyword(index) {
        deleteTaskListItem('filter.include', index, 'Delete this include keyword?');
    }
    function addExcludeKeyword(ul, keyword) {
        // Ensure filter and exclude array exist before adding
        const context = getCurrentTaskContext();
        if (!context) return;
        const { taskName, taskConfig, isNew } = context;
        const targetConfig = taskConfig || (isNew ? currentTasks[taskName] : null);

        if (targetConfig?.filter) {
            if (!targetConfig.filter.exclude) targetConfig.filter.exclude = [];
            addTaskListItem('filter.exclude', keyword);
        } else {
            console.warn(`addExcludeKeyword: Filter section does not exist for task "${taskName}". Cannot add keyword.`);
        }
    }
    function deleteExcludeKeyword(index) {
        deleteTaskListItem('filter.exclude', index, 'Delete this exclude keyword?');
    }


    function renderExtracterSection(form, extracter) {
        const section = createFormSection(form, 'Extracter');
        if (!extracter) {
            const addButton = document.createElement('button');
            addButton.type = 'button';
            addButton.textContent = 'Add Extracter';
            addButton.classList.add('button', 'secondary-button');
            addButton.addEventListener('click', () => {
                // Use generic toggle function
                toggleTaskSection('extracter', { tag: 'link', pattern: '' });
            });
            section.appendChild(addButton);
            return;
        }

        const validTags = ['title', 'link', 'description', 'enclosure', 'guid'];
        const tagOptions = validTags.map(tag => ({ value: tag, text: tag }));
        createSelectField(section, 'extracterTag', 'Tag', tagOptions, extracter.tag);
        createTextField(section, 'extracterPattern', 'Pattern (Regex)', extracter.pattern || '');

        const removeExtracterBtn = document.createElement('button');
        removeExtracterBtn.type = 'button';
        removeExtracterBtn.textContent = 'Remove Extracter Section';
        removeExtracterBtn.classList.add('button', 'danger-button');
        removeExtracterBtn.id = "removeExtracterBtn";
        // removeExtracterBtn.style.marginTop = '10px';
        removeExtracterBtn.addEventListener('click', () => {
            // Use generic toggle function
            toggleTaskSection('extracter', null, 'Are you sure you want to remove the extracter section?');
        });
        section.appendChild(removeExtracterBtn);
    }
    // --- Generic Task Modification Helpers ---

    // Helper to get nested property value using dot notation string
    function getNestedProperty(obj, path) {
        // Handle cases where obj is null/undefined early
        if (!obj) return undefined;
        return path.split('.').reduce((acc, part) => acc && acc[part], obj);
    }

    // Helper to set nested property value using dot notation string
    // Creates path if it doesn't exist
    function setNestedProperty(obj, path, value) {
        const parts = path.split('.');
        let current = obj;
        for (let i = 0; i < parts.length - 1; i++) {
            const part = parts[i];
            // Create object or array if path segment doesn't exist or is not the correct type
            if (current[part] === undefined || current[part] === null || typeof current[part] !== 'object') {
                // Look ahead to see if the next part implies an array index (though we don't use indices in paths here)
                // For simplicity, always create objects. If an array is needed, it should be initialized explicitly.
                current[part] = {};
            }
            current = current[part];
        }
        // Ensure the final part target exists before setting the value
        if (typeof current === 'object' && current !== null) {
            current[parts[parts.length - 1]] = value;
        } else {
            console.error(`setNestedProperty: Cannot set property on non-object at path "${path}"`);
        }
    }


    // Generic function to add an item to a list within the current task config
    function addTaskListItem(itemPath, value) {
        const context = getCurrentTaskContext();
        if (!context) return;
        const { taskName, isNew, taskConfig } = context;

        // Use taskConfig for existing tasks, or the temporary currentTasks[taskName] for new ones
        const targetConfig = taskConfig || (isNew ? currentTasks[taskName] : null);
        if (!targetConfig) {
            console.error(`addTaskListItem: Could not find target config for task "${taskName}" (isNew: ${isNew})`);
            return;
        }

        let list = getNestedProperty(targetConfig, itemPath);

        // Ensure the list exists and is an array
        if (!Array.isArray(list)) {
            console.warn(`addTaskListItem: Path "${itemPath}" did not resolve to an array. Initializing.`);
            // Attempt to initialize the path up to the array
            const pathParts = itemPath.split('.');
            const arrayName = pathParts.pop();
            const parentPath = pathParts.join('.');
            let parentObj = targetConfig;
            if (parentPath) {
                parentObj = getNestedProperty(targetConfig, parentPath);
                if (!parentObj || typeof parentObj !== 'object') {
                    // Need to create the parent structure
                    setNestedProperty(targetConfig, parentPath, {});
                    parentObj = getNestedProperty(targetConfig, parentPath);
                }
            }
            if (!parentObj || typeof parentObj !== 'object') {
                console.error(`addTaskListItem: Could not create/find parent object for path "${itemPath}"`);
                return;
            }
            // Initialize the array
            parentObj[arrayName] = [];
            list = parentObj[arrayName];
            // Ensure the list is set back onto the main config object if nested path creation happened
            setNestedProperty(targetConfig, itemPath, list); // Ensure the newly created array is set
        }


        list.push(value);
        renderTaskDetail(taskName);
    }

    // Generic function to delete an item from a list within the current task config
    function deleteTaskListItem(itemPath, index, confirmMessage) {
        const context = getCurrentTaskContext();
        if (!context) return;
        const { taskName, isNew, taskConfig } = context;

        const targetConfig = taskConfig || (isNew ? currentTasks[taskName] : null);
        if (!targetConfig) {
            console.error(`deleteTaskListItem: Could not find target config for task "${taskName}" (isNew: ${isNew})`);
            return;
        }

        const list = getNestedProperty(targetConfig, itemPath);

        if (!Array.isArray(list)) {
            console.error(`deleteTaskListItem: Path "${itemPath}" did not resolve to an array.`);
            return;
        }

        if (index < 0 || index >= list.length) {
            console.error(`deleteTaskListItem: Invalid index ${index} for path "${itemPath}".`);
            return;
        }

        if (confirm(confirmMessage || `Are you sure you want to delete this item?`)) {
            list.splice(index, 1);
            renderTaskDetail(taskName);
        }
    }

    // Generic function to add or remove an entire section (object property) from the task config
    function toggleTaskSection(sectionName, defaultData = null, confirmMessage = null) {
        const context = getCurrentTaskContext();
        if (!context) return;
        const { taskName, isNew, taskConfig } = context;

        const targetConfig = taskConfig || (isNew ? currentTasks[taskName] : null);
        if (!targetConfig) {
            console.error(`toggleTaskSection: Could not find target config for task "${taskName}" (isNew: ${isNew})`);
            return;
        }

        if (defaultData !== null) {
            // Add section
            targetConfig[sectionName] = defaultData;
            renderTaskDetail(taskName);
        } else {
            // Remove section
            if (confirmMessage && !confirm(confirmMessage)) {
                return;
            }
            delete targetConfig[sectionName];
            renderTaskDetail(taskName);
        }
    }


    // --- Form Submission and Deletion ---

    async function handleFormSubmit(event) { // Keep async as it calls apiFetch
        event.preventDefault();
        const form = event.target;
        // Use helper function to get context
        const context = getCurrentTaskContext();
        if (!context) {
            console.error("handleFormSubmit: Failed to get task context.");
            return;
        }
        const { taskName, isNew: isNewTask, taskConfig: currentConfig } = context;

        // Construct the final TaskConfig object to be sent to the API
        // Start with the current state from `currentTasks` which reflects UI changes made via helpers.
        // Use deep clone to avoid modifying the original object directly before API call success.
        const baseConfig = currentConfig || (isNewTask ? currentTasks[taskName] : null);
        if (!baseConfig) {
            console.error(`handleFormSubmit: Base config not found for task "${taskName}" (isNew: ${isNewTask})`);
            alert('Internal Error: Could not prepare task data for saving.');
            return;
        }
        const finalTaskConfig = JSON.parse(JSON.stringify(baseConfig));


        // Update basic fields directly from the form
        finalTaskConfig.interval = parseInt(form.querySelector('#interval')?.value, 10) || 10;

        // Update Extracter directly from form fields if the section exists
        const extracterTagEl = form.querySelector('#extracterTag');
        const extracterPatternEl = form.querySelector('#extracterPattern');
        if (extracterTagEl && extracterPatternEl) {
            const pattern = extracterPatternEl.value.trim();
            if (pattern) {
                finalTaskConfig.extracter = {
                    tag: extracterTagEl.value,
                    pattern: pattern
                };
            } else {
                // Remove extracter if pattern is empty
                delete finalTaskConfig.extracter;
            }
        } else {
            // Ensure extracter is removed if the section wasn't rendered
            delete finalTaskConfig.extracter;
        }


        // Clean up Filter: remove empty include/exclude arrays and the filter object itself if empty
        if (finalTaskConfig.filter) {
            if (finalTaskConfig.filter.include?.length === 0) delete finalTaskConfig.filter.include;
            if (finalTaskConfig.filter.exclude?.length === 0) delete finalTaskConfig.filter.exclude;
            if (Object.keys(finalTaskConfig.filter).length === 0) delete finalTaskConfig.filter;
        }

        // Clean up Feed: Ensure it's not just an empty object if URLs were deleted
        if (finalTaskConfig.feed && (!finalTaskConfig.feed.URLs || finalTaskConfig.feed.URLs.length === 0)) {
            delete finalTaskConfig.feed;
        }


        delete finalTaskConfig.isTemporary;


        // --- Basic Validation (using finalTaskConfig) ---
        if (!finalTaskConfig.downloaders || finalTaskConfig.downloaders.length === 0) {
            alert('Task must have at least one downloader.');
            return;
        }
        if (!finalTaskConfig.feed || !finalTaskConfig.feed.URLs || finalTaskConfig.feed.URLs.length === 0) {
            alert('Task must have at least one feed URL.');
            return;
        }

        console.log("Submitting Task Config:", JSON.stringify(finalTaskConfig, null, 2));

        try {
            let result;
            if (isNewTask) {
                // Task name should already be validated and set in context
                if (!taskName) {
                    alert('Internal Error: New task name is missing during submission.');
                    return;
                }
                // Use POST for new task
                result = await apiFetch('/api/tasks', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name: taskName, config: finalTaskConfig })
                });
                alert(`Task "${taskName}" created successfully!`);
                selectedTaskName = taskName;
            } else {
                // Use PUT for existing task
                result = await apiFetch(`/api/tasks/${taskName}`, {
                    method: 'PUT',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify(finalTaskConfig)
                });
                alert(`Task "${taskName}" updated successfully!`);
            }
            console.log("API Response:", result);
            // await loadTasks();
            // Re-select the task after reload
            if (selectedTaskName && currentTasks[selectedTaskName]) {
                delete currentTasks[selectedTaskName].isNew;
                selectTask(selectedTaskName);
            } else {
                clearTaskDetailPanel();
            }

        } catch (error) {
            // Error already handled by apiFetch
            console.error("Failed to save task:", error);
        }
    }

    async function deleteTask(taskName) {
        if (confirm(`Are you sure you want to delete the task "${taskName}"? This cannot be undone.`)) {
            try {
                const result = await apiFetch(`/api/tasks/${taskName}`, { method: 'DELETE' });
                console.log("Delete Response:", result);
                alert(`Task "${taskName}" deleted successfully!`);
                selectedTaskName = null;
                await loadTasks();
                clearTaskDetailPanel();
            } catch (error) {
                console.error("Failed to delete task:", error);
            }
        }
    }

    // --- Add New Task ---
    function showNewTaskForm() {
        selectedTaskName = null;
        renderTaskList();
        taskFormContainer.innerHTML = '';

        const form = document.createElement('form');
        form.id = 'task-form';
        form.dataset.isNew = 'true';

        // --- Basic Info ---
        const infoSection = createFormSection(form, 'New Task Info');
        const newTaskNameInput = createTextField(infoSection, 'newTaskName', 'Task Name *', '', false);
        newTaskNameInput.addEventListener('change', function (event) {
            const newTaskNameValue = event.target.value.trim();
            const existingTaskNames = Object.keys(currentTasks);
            const createButton = form.querySelector('button[type="submit"]');

            // Clear previous task name from dataset and potentially from currentTasks
            const previousTaskName = form.dataset.taskName;
            if (previousTaskName && previousTaskName !== newTaskNameValue && currentTasks[previousTaskName]?.isTemporary) {
                delete currentTasks[previousTaskName];
            }
            form.removeAttribute('data-task-name');

            if (newTaskNameValue === '') {
                newTaskNameInput.setCustomValidity('Task name cannot be empty.');
                if (createButton) createButton.disabled = true;
            } else if (existingTaskNames.includes(newTaskNameValue)) {
                newTaskNameInput.setCustomValidity(`Task name "${newTaskNameValue}" already exists.`);
                if (createButton) createButton.disabled = true;
            } else {
                newTaskNameInput.setCustomValidity('');
                form.dataset.taskName = newTaskNameValue;
                // Create the temporary task object if it doesn't exist
                if (!currentTasks[newTaskNameValue]) {
                    currentTasks[newTaskNameValue] = {
                        interval: parseInt(form.querySelector('#interval')?.value, 10) || 10, // Get interval from form or default
                        downloaders: [],
                        feed: { URLs: [] }, // Initialize feed structure
                        filter: null,
                        extracter: null,
                        isTemporary: true,
                        isNew: true
                    };
                    // console.log(`Created temporary task object for: ${newTaskNameValue}`);
                }
                if (createButton) createButton.disabled = false;
            }
            newTaskNameInput.reportValidity();
        });
        // Initially disable create button until a valid name is entered
        const initialCreateButton = form.querySelector('button[type="submit"]');
        if (initialCreateButton) initialCreateButton.disabled = true;
        createNumberField(infoSection, 'interval', 'Fetch Interval (minutes)', 10);

        // --- Initialize empty sections for user to add items ---
        renderDownloaderSection(form, []);
        renderFeedSection(form, []);
        renderFilterSection(form, null);
        renderExtracterSection(form, null);


        // --- Action Buttons ---
        const actionDiv = document.createElement('div');
        actionDiv.classList.add('action-buttons');

        const createButton = document.createElement('button');
        createButton.type = 'submit';
        createButton.textContent = 'Create Task';
        createButton.classList.add('button', 'primary-button');
        actionDiv.appendChild(createButton);

        const cancelButton = document.createElement('button');
        cancelButton.type = 'button';
        cancelButton.textContent = 'Cancel';
        cancelButton.classList.add('button', 'secondary-button');
        cancelButton.addEventListener('click', clearTaskDetailPanel);
        actionDiv.appendChild(cancelButton);


        form.appendChild(actionDiv);
        taskFormContainer.appendChild(form);

        form.addEventListener('submit', handleFormSubmit);
        newTaskNameInput.focus();
    }


    // --- Modal Management ---
    function openModal(title, contentGenerator) {
        modalTitle.textContent = title;
        modalBody.innerHTML = ''; // Clear previous content
        contentGenerator(modalBody); // Populate modal body
        modal.style.display = 'block';
    }

    function closeModal() {
        modal.style.display = 'none';
        modalBody.innerHTML = ''; // Clear content
    }

    // Modal for Downloader Add/Edit
    function openDownloaderModal(downloaderData = null, index = null) {
        const isEditing = downloaderData !== null && index !== null;
        const title = isEditing ? `Edit Downloader #${index + 1}` : 'Add New Downloader';

        openModal(title, (body) => {
            const modalForm = document.createElement('form');
            modalForm.id = 'downloader-modal-form';

            // Type Select
            const typeOptions = [{ value: 'aria2c', text: 'Aria2c' }, { value: 'transmission', text: 'Transmission' }];
            const typeSelect = createSelectField(modalForm, 'dlType', 'Type *', typeOptions, downloaderData?.type || 'aria2c');

            // Common Fields
            const hostInput = createTextField(modalForm, 'dlHost', 'Host', downloaderData?.host || '', false, "localhost");

            // Type-Specific Fields (initially hidden)
            const aria2cFields = document.createElement('div');
            aria2cFields.id = 'aria2c-fields';
            const aria2cPortInput = createNumberField(aria2cFields, 'dlAria2cPort', 'Port', downloaderData?.port || '', false, 6800);
            const aria2cRpcPathInput = createTextField(aria2cFields, 'dlAria2cRpcPath', 'RPC Path', downloaderData?.rpcPath || '', false, "/jsonrpc");
            const tokenInput = createTextField(aria2cFields, 'dlToken', 'Token', downloaderData?.token || '');
            modalForm.appendChild(aria2cFields);

            const transmissionFields = document.createElement('div');
            transmissionFields.id = 'transmission-fields';
            const transPortInput = createNumberField(transmissionFields, 'dlTransPort', 'Port', downloaderData?.port || '', false, 9091);
            const transRpcPathInput = createTextField(transmissionFields, 'dlTransRpcPath', 'RPC Path', downloaderData?.rpcPath || '', false, "/transmission/rpc");
            const usernameInput = createTextField(transmissionFields, 'dlUsername', 'Username', downloaderData?.username || '');
            const passwordInput = createPasswordField(transmissionFields, 'dlPassword', 'Password', downloaderData?.password || '');
            modalForm.appendChild(transmissionFields);

            // Common Fields
            const useHttpsCheckbox = createCheckboxField(modalForm, 'dlUseHttps', 'Use HTTPS', downloaderData?.useHttps || false);
            const autoCleanUpCheckbox = createCheckboxField(modalForm, 'dlAutoCleanUp', 'Auto CleanUp', downloaderData?.autoCleanUp || false);

            // Function to toggle visibility based on selected type
            const toggleSpecificFields = () => {
                const selectedType = typeSelect.value;
                aria2cFields.style.display = selectedType === 'aria2c' ? 'block' : 'none';
                transmissionFields.style.display = selectedType === 'transmission' ? 'block' : 'none';
            };

            typeSelect.addEventListener('change', toggleSpecificFields);
            toggleSpecificFields();

            // Save Button
            const saveBtn = document.createElement('button');
            saveBtn.type = 'submit';
            saveBtn.textContent = isEditing ? 'Save Downloader Changes' : 'Add Downloader';
            saveBtn.classList.add('button', 'primary-button');
            modalForm.appendChild(saveBtn);

            modalForm.addEventListener('submit', (e) => {
                e.preventDefault();
                const newDownloader = {
                    type: typeSelect.value,
                    host: hostInput.value.trim() || undefined, // Use undefined for empty optional fields
                    useHttps: useHttpsCheckbox.checked || undefined,
                    autoCleanUp: autoCleanUpCheckbox.checked || undefined,
                };
                // Add type-specific fields
                if (newDownloader.type === 'aria2c') {
                    newDownloader.port = parseInt(aria2cPortInput.value, 10) || undefined,
                        newDownloader.rpcPath = aria2cRpcPathInput.value.trim() || undefined,
                        newDownloader.token = tokenInput.value.trim() || undefined;
                } else if (newDownloader.type === 'transmission') {
                    newDownloader.port = parseInt(transPortInput.value, 10) || undefined,
                        newDownloader.rpcPath = transRpcPathInput.value.trim() || undefined,
                        newDownloader.username = usernameInput.value.trim() || undefined;
                    newDownloader.password = passwordInput.value || undefined; // Keep empty password if entered
                }

                // Remove undefined fields before saving (optional, backend might handle defaults)
                Object.keys(newDownloader).forEach(key => {
                    if (newDownloader[key] === undefined || newDownloader[key] === null || newDownloader[key] === '') {
                        // Let's keep empty strings for now, backend defaults should handle it.
                        // delete newDownloader[key];
                        // Exception: keep useHttps: false and autoCleanUp: false if explicitly unchecked
                        if (key === 'useHttps' && !newDownloader[key]) newDownloader[key] = false;
                        if (key === 'autoCleanUp' && !newDownloader[key]) newDownloader[key] = false;
                        // Keep empty token/user/pass if needed
                        if (key === 'token' && newDownloader.type === 'aria2c' && newDownloader[key] === '') newDownloader[key] = '';
                        if (key === 'username' && newDownloader.type === 'transmission' && newDownloader[key] === '') newDownloader[key] = '';
                        if (key === 'password' && newDownloader.type === 'transmission' && newDownloader[key] === '') newDownloader[key] = '';

                        // Remove truly empty optional fields if desired
                        if (newDownloader[key] === '' && ['host', 'port', 'rpcPath'].includes(key)) {
                            delete newDownloader[key];
                        }
                        if (key === 'port' && isNaN(newDownloader[key])) delete newDownloader[key];
                    }
                });

                // Use generic functions to add/update the downloader
                const targetPath = 'downloaders';
                if (isEditing) {
                    // Need a way to update item at index, generic add/delete aren't enough
                    // For now, directly modify and re-render
                    const contextForUpdate = getCurrentTaskContext(); // Re-fetch context in case it changed
                    if (!contextForUpdate) return;
                    const targetConfig = contextForUpdate.taskConfig || (contextForUpdate.isNew ? currentTasks[contextForUpdate.taskName] : null);
                    if (targetConfig && Array.isArray(targetConfig.downloaders) && index < targetConfig.downloaders.length) {
                        targetConfig.downloaders[index] = newDownloader;
                        renderTaskDetail(contextForUpdate.taskName);
                    } else {
                        alert(`Error: Could not update downloader at index ${index}.`);
                    }
                } else {
                    addTaskListItem(targetPath, newDownloader);
                }

                closeModal();
            });

            body.appendChild(modalForm);
        });
    }


    // --- Event Listeners ---
    addTaskBtn.addEventListener('click', showNewTaskForm);
    closeModalBtn.addEventListener('click', closeModal);
    // Close modal if clicking outside the content
    window.addEventListener('click', (event) => {
        if (event.target == modal) {
            closeModal();
        }
    });


    // --- Initial Load ---
    loadTasks();

});
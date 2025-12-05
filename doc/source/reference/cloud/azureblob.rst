==========
Azure Blob
==========

Waarp Gateway permet d'utiliser un bucket Azure Blob à la place du disque dur local.

.. note::
   Ne pas confondre avec :doc:`Azure Files<azurefiles>`.

Configuration
-------------

Pour utiliser une instance cloud, celle-ci doit d'abord être créée et configurée.
Pour créer une instance cloud Azure Blob, le type renseigné doit être ``azblob``
ou ``azureblob``.

Authentification
^^^^^^^^^^^^^^^^

Pour se connecter à une instance Azure, Waarp Gateway a besoin d'identifiants.
Ces identifiants peuvent être fournis de 2 manières différentes :

- Soit en les renseignant directement à la création de l'instance cloud, auquel cas :

  - Le paramètre *key* doit contenir le nom de compte
  - Le paramètre *secret* doit contenir la clé du compte
- Soit en les renseignant dans des variables d'environnement. Les variables
  suivantes doivent être renseignées :

  - :envvar:`AZURE_STORAGE_ACCOUNT` pour le nom de compte
  - :envvar:`AZURE_CLIENT_SECRET` pour la clé d'authentification
  - :envvar:`AZURE_TENANT_ID` pour l'ID du locataire Microsoft Entra
  - :envvar:`AZURE_CLIENT_ID` pour l'ID de l'application

Options
^^^^^^^

Les options de configuration suivantes sont disponibles pour Azure Blob :

* **endpoint**: L'adresse du serveur Azure (si celle-ci est différente de l'adresse
  par défaut ``core.windows.net``).
* **env_auth**: Booléen, mettre à ``true`` pour activer l'authentification via
  variables d'environnement décrite ci-dessus.

Exemple
-------

Prenons le cas de figure suivant :

- fichier: ``doc/waarp-gateway.pdf``
- authentification via variables d'environnement

Dans un premier temps, l'instance cloud doit être définie. Dans cet exemple, nous
lui donnerons le nom "ex-azblob".

La commande de création pour cette instance cloud est donc :

.. code-block:: shell
   waarp-gateway cloud add -n "ex-azblob" -t "azblob" -o "env_auth:true"

Par la suite, lors de mon transfert, le chemin du fichier devra donc ressembler à :

| ex-azblob:doc/waarp-gateway.pdf


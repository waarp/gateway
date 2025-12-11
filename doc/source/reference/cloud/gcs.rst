============
Google Cloud
============

Waarp Gateway permet d'utiliser Google Cloud Storage (GCS) à la place du disque
dur local.

Configuration
-------------

Pour utiliser une instance cloud, celle-ci doit d'abord être créée et configurée.
Pour créer une instance Google Cloud, le type renseigné doit être ``gcs`` ou
``google``.

Authentification
^^^^^^^^^^^^^^^^

Pour se connecter à une instance Google Cloud, Waarp Gateway a besoin d'un token
JWT pour OAuth2. Ce token peut être fournis de 2 manières différentes :

- Soit en le renseignant directement à la création de l'instance cloud via
  le paramètre *secret* de l'instance cloud.
- Soit en le stockant sur le disque local, puis en donnant à Gateway le chemin
  de ce fichier. Le chemin peut être donné :

  - soit via la variable d'environnement :envvar:`GOOGLE_APPLICATION_CREDENTIALS`
  - soit via le paramètre *key* à la création de l'instance cloud.

Options
^^^^^^^

Les options de configuration suivantes sont disponibles pour GCS sont :

* **bucket**: *REQUIS* - Le nom du bucket à accéder.
* **project_number**: *REQUIS* - Le numéro du projet Google Cloud.
* **env_auth**: Booléen, mettre à ``true`` pour activer l'authentification via
  variable d'environnement décrite ci-dessus.
* **bucket_policy_only**: Booléen, mettre à ``true`` si le bucket utilise une
  politique d'accès uniforme. Mettre à ``false`` si le bucket utilise des listes
  de contrôle d'accès.


Exemple
-------

Prenons le cas de figure suivant :

- fichier: ``doc/waarp-gateway.pdf``
- bucket: ``archive``
- authentification via variables d'environnement
- politique d'accès uniforme

Dans un premier temps, l'instance cloud doit être définie. Dans cet exemple, nous
lui donnerons le nom "ex-gcs".

La commande de création pour cette instance cloud est donc :

.. code-block:: shell

   waarp-gateway cloud add -n "ex-gcs" -t "gcs" -o "env_auth:true" -o "bucket:archive" -o "bucket_policy_only:true"

Par la suite, lors de mon transfert, le chemin du fichier devra donc ressembler à :

| ex-gcs:doc/waarp-gateway.pdf


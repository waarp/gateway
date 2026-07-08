Ajouter un filewatcher
======================

.. http:post:: /api/filewatchers

   Ajoute un nouveau *filewatcher*.

   :reqheader Authorization: Les identifiants de l'utilisateur REST.

   :reqjson string flow: Le nom du flux auquel le *filewatcher* appartient.
   :reqjson bool disabled: Indique si le *filewatcher* est désactivé au démarrage.
   :reqjson string interval: La fréquence à laquelle le *filewatcher* interrogera
      le partenaire distant pour obtenir la liste des fichiers à récupérer.
   :reqjson string pattern: Le pattern de fichier à matcher (format
      `glob<https://en.wikipedia.org/wiki/Glob_(programming)>`_.
   :reqjson bool noDuplicateCheck: Indique si la détection de doublons est désactivée.
      Par défaut, le *filewatcher* ignore les fichiers qui ont déjà été récupérés
      lors d'un précédent passage. Mettre à *true* désactive cette vérification.
   :reqjson string partner: Le partenaire interrogé par le *filewatcher*.
   :reqjson string account: Le compte utilisé pour l'authentification.
   :reqjson string client: Le client utilisé pour la requête.
   :reqjson string rule: La règle à utiliser pour les transferts.

   :statuscode 201: Le *filewatcher* a été créé avec succès
   :statuscode 400: Requête invalide
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action

   :resheader Location: Le chemin d'accès au nouveau *filewatcher* créé

   |

   **Exemple de requête**

      .. code-block:: http

         POST https://my_waarp_gateway.net/api/filewatchers HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==
         Content-Type: application/json
         Content-Length: 195

         {
           "flow": "my-filewatcher",
           "disabled": false,
           "interval": "5m",
           "pattern": "*.txt",
           "noDuplicateCheck": false,
           "partner": "my-partner",
           "account": "my-account",
           "client": "my-client",
           "rule": "my-rule"
         }

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 201 CREATED
         Location: https://my_waarp_gateway.net/api/filewatchers/my-filewatcher

Consulter un filewatcher
========================

.. http:get:: /api/filewatchers/(string:flow)

   Renvoie le *filewatcher* demandé.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :resjson string flow: Le nom du flux auquel le *filewatcher* appartient.
   :resjson bool disabled: Indique si le *filewatcher* est désactivé au démarrage.
   :resjson string interval: La fréquence à laquelle le *filewatcher* interrogera
      le partenaire distant pour obtenir la liste des fichiers à récupérer.
   :resjson string pattern: Le pattern de fichier à matcher (format
      `glob<https://en.wikipedia.org/wiki/Glob_(programming)>`_.
   :resjson bool noDuplicateCheck: Indique si la détection de doublons est désactivée.
      Par défaut, le *filewatcher* ignore les fichiers qui ont déjà été récupérés
      lors d'un précédent passage. Mettre à *true* désactive cette vérification.
   :resjson string partner: Le partenaire interrogé par le *filewatcher*.
   :resjson string account: Le compte utilisé pour l'authentification.
   :resjson string client: Le client utilisé pour la requête.
   :resjson string rule: La règle à utiliser pour les transferts.

   :statuscode 200: Le *filewatcher* a été renvoyé avec succès
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action
   :statuscode 404: Le *filewatcher* demandé n'existe pas

   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/filewatchers/my-filewatcher HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 197

         {
           "flow": "my-filewatcher",
           "disabled": false,
           "interval": "1h15m30s",
           "pattern": "*.txt",
           "noDuplicateCheck": false,
           "partner": "my-partner",
           "account": "my-account",
           "client": "my-client",
           "rule": "my-rule"
         }

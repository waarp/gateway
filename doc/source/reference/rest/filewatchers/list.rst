Lister les filewatchers
=======================

.. http:get:: /api/filewatchers

   Renvoie une liste des *filewatchers* connus.

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sort: Le paramètre selon lequel les *filewatchers* seront triés *(défaut: flow+)*
   :type sort: [flow+\|flow-]

   :resjson array filewatchers: La liste des *filewatchers* demandés
   :resjsonarr string flow: Le nom du flux auquel le *filewatcher* appartient.
   :resjsonarr bool disabled: Indique si le *filewatcher* est désactivé au démarrage.
   :resjsonarr string interval: La fréquence à laquelle le *filewatcher* interrogera
      le partenaire distant pour obtenir la liste des fichiers à récupérer.
   :resjsonarr string pattern: Le pattern de fichier à matcher (format
      `glob<https://en.wikipedia.org/wiki/Glob_(programming)>`_.
   :resjsonarr bool noDuplicateCheck: Indique si la détection de doublons est désactivée.
      Par défaut, le *filewatcher* ignore les fichiers qui ont déjà été récupérés
      lors d'un précédent passage. Mettre à *true* désactive cette vérification.
   :resjsonarr string partner: Le partenaire interrogé par le *filewatcher*.
   :resjsonarr string account: Le compte utilisé pour l'authentification.
   :resjsonarr string client: Le client utilisé pour la requête.
   :resjsonarr string rule: La règle à utiliser pour les transferts.

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Requête invalide
   :statuscode 401: Authentification REST invalide
   :statuscode 403: L'utilisateur REST n'a pas le droit d'effectuer cette action

   |

   **Exemple de requête**

      .. code-block:: http

         GET https://my_waarp_gateway.net/api/filewatchers?limit=10&sort=flow+ HTTP/1.1
         Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==

   **Exemple de réponse**

      .. code-block:: http

         HTTP/1.1 200 OK
         Content-Type: application/json
         Content-Length: 213

         {
           "filewatchers": [{
             "flow": "my-filewatcher",
             "disabled": false,
             "interval": "5m",
             "pattern": "*.txt",
             "noDuplicateCheck": false,
             "partner": "my-partner",
             "account": "my-account",
             "client": "my-client",
             "rule": "my-rule"
           }]
         }

Lister les règles
=================

.. http:get:: /api/rules

   Renvoie une liste des règles emplissant les critères donnés en paramètres
   de requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sortby: Le paramètre selon lequel les règles seront triées *(défaut: name)*
   :type sortby: [name]
   :param order: L'ordre dans lequel les règles sont triées *(défaut: asc)*
   :type order: [asc|desc]

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/rules?limit=5 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide

   :resjson array rules: La liste des règles demandées
   :resjsonarr number id: L'identifiant unique de la règle
   :resjsonarr string name: Le nom de la règle
   :resjsonarr string comment: Un commentaire optionnel à propos de la règle (description...)
   :resjsonarr bool isSend: Si vrai, la règle peut être utilisée lors de l'envoi de fichiers,
                         si faux, la règle peut être utilisée lors de la réception de fichiers
   :reqjson string path: Le chemin de destination du fichier

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 453

          {
            "rules": [{
              "id": 1,
              "name": "règle exemple envoi",
              "comment": "ceci est un exemple de règle d'envoi",
              "isSend": true
              "path": "/chemin/distant/de/destination/du/fichier"
            },{
              "id": 2,
              "name": "règle exemple réception",
              "comment": "ceci est un exemple de règle de réception",
              "isSend": false
              "path": "/chemin/local/de/destination/du/fichier"
            }]
          }
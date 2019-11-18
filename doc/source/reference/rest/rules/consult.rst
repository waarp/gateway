Consulter une règle
===================

.. http:get:: /api/rules/(int:rule_id)

   Renvoie la règle portant le numéro ``rule_id``.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   **Exemple de requête**

       .. code-block:: http

          GET https://my_waarp_gateway.net/api/rules/1 HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: Lea règle a été renvoyée avec succès
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: La règle demandée n'existe pas

   :resjson number id: L'identifiant unique de la règle
   :resjson string name: Le nom de la règle
   :resjson string comment: Un commentaire optionnel à propos de la règle (description...)
   :resjson bool isSend: Si vrai, la règle peut être utilisée lors de l'envoi de fichiers,
                         si faux, la règle peut être utilisée lors de la réception de fichiers
   :reqjson string path: Le chemin de destination du fichier

   **Exemple de réponse**

       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 137

          {
            "id": 1,
            "name": "règle exemple envoi",
            "comment": "ceci est un exemple de règle d'envoi",
            "isSend": true
            "path": "/chemin/distant/de/destination/du/fichier"
          }
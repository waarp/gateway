Lister les comptes
==================

.. http:get:: /api/partners/(partner)/accounts

   Renvoie une liste des comptes rattaché au partenaire nommé `partner` remplissant
   les critères données en paramètres de requête.

   **Requête**

   :reqheader Authorization: Les identifiants de l'utilisateur

   :param limit: Le nombre maximum de résultats souhaités *(défaut: 20)*
   :type limit: int
   :param offset: Le numéro du premier résultat souhaité *(défaut: 0)*
   :type offset: int
   :param sortby: Le paramètre selon lequel les comptes seront triés *(défaut: name)*
   :type sortby: [name]
   :param order: L'ordre dans lequel les compte sont triés *(défaut: asc)*
   :type order: [asc|desc]

   :Example:
       .. code-block:: http

          GET /api/partners/partenaire1/accounts?limit=10&order=desc HTTP/1.1
          Authorization: Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==


   **Réponse**

   :statuscode 200: La liste a été renvoyée avec succès
   :statuscode 400: Un ou plusieurs des paramètres de requêtes sont invalides
   :statuscode 401: Authentification d'utilisateur invalide
   :statuscode 404: Le partenaire demandé n'existe pas

   :Response JSON Object:
       * **Accounts** (*array* of *object*) - La liste des comptes demandés

           * **Username** (*string*) - Le nom d'utilisateur du compte

   :Example:
       .. code-block:: http

          HTTP/1.1 200 OK
          Content-Type: application/json
          Content-Length: 79

          {
            "Accounts": [{
              "Name": "partenaire2",
            },{
              "Name": "partenaire1",
            }]
          }